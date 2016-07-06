package domain

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/pivotal-golang/lager"
)

type UrlGetter interface {
	Get(url string) (*http.Response, error)
}

func HttpUrlGetterProvider(healthcheckTimeout time.Duration) UrlGetter {
	return &http.Client{
		Timeout: healthcheckTimeout,
	}
}

var UrlGetterProvider = HttpUrlGetterProvider

//go:generate counterfeiter . Cluster
type Cluster interface {
	Monitor() chan<- interface{}
	RouteToBackend(clientConn net.Conn) error
	AsJSON() ClusterJSON
	EnableTraffic()
	DisableTraffic(message string)
}

type cluster struct {
	mutex               sync.RWMutex
	backends            Backends
	currentBackendIndex int
	logger              lager.Logger
	healthcheckTimeout  time.Duration
	arpManager          ArpManager
	message             string
}

func NewCluster(backends Backends, healthcheckTimeout time.Duration, logger lager.Logger, arpManager ArpManager) Cluster {
	return &cluster{
		backends:            backends,
		currentBackendIndex: 0,
		logger:              logger,
		healthcheckTimeout:  healthcheckTimeout,
		arpManager:          arpManager,
	}
}

func (c cluster) Monitor() chan<- interface{} {
	client := UrlGetterProvider(c.healthcheckTimeout)
	stopChan := make(chan interface{})
	for backend := range c.backends.All() {
		c.monitorHealth(backend, client, stopChan)
	}
	return stopChan
}

func (c cluster) RouteToBackend(clientConn net.Conn) error {
	activeBackend := c.backends.Active()
	if activeBackend == nil {
		return errors.New("No active Backend")
	}
	return activeBackend.Bridge(clientConn)
}

func (c cluster) monitorHealth(backend Backend, client UrlGetter, stopChan <-chan interface{}) {
	go func() {
		counters := c.setupCounters()
		for {
			select {
			case <-time.After(c.healthcheckTimeout / 5):
				c.dialHealthcheck(backend, client, counters)
			case <-stopChan:
				return
			}
		}
	}()
}

func (c cluster) setupCounters() *DecisionCounters {
	counters := NewDecisionCounters()
	logFreq := uint64(5)
	clearArpFreq := uint64(5)

	//used to make logs less noisy
	counters.AddCondition("log", func() bool {
		return (counters.GetCount("dial") % logFreq) == 0
	})

	//only clear ARP cache after X consecutive unhealthy dials
	counters.AddCondition("clearArp", func() bool {
		// golang makes it difficult to tell whether the value of an interface is nil
		if reflect.ValueOf(c.arpManager).IsNil() {
			return false
		} else {
			checks := counters.GetCount("consecutiveUnhealthyChecks")
			return (checks > 0) && (checks%clearArpFreq) == 0
		}
	})

	return counters
}

func (c cluster) dialHealthcheck(backend Backend, client UrlGetter, counters *DecisionCounters) {

	counters.IncrementCount("dial")
	shouldLog := counters.Should("log")

	url := backend.HealthcheckUrl()
	resp, err := client.Get(url)

	if err == nil && resp.StatusCode == http.StatusOK {
		c.backends.SetHealthy(backend)
		counters.ResetCount("consecutiveUnhealthyChecks")

		if shouldLog {
			c.logger.Debug("Healthcheck succeeded", lager.Data{"endpoint": url})
		}
	} else {
		c.backends.SetUnhealthy(backend)
		counters.IncrementCount("consecutiveUnhealthyChecks")

		if shouldLog {
			c.logger.Error(
				"Healthcheck failed on backend",
				fmt.Errorf("Non-200 status code from healthcheck"),
				lager.Data{
					"backend":  backend.AsJSON(),
					"endpoint": url,
					"resp":     fmt.Sprintf("%#v", resp),
					"err":      err,
				},
			)
		}
	}

	if counters.Should("clearArp") {
		backendHost := backend.AsJSON().Host

		if c.arpManager.IsCached(backendHost) {
			err = c.arpManager.ClearCache(backendHost)
			if err != nil {
				c.logger.Error("Failed to clear arp cache", err)
			}
		}
	}
}

type ClusterJSON struct {
	CurrentBackendIndex uint   `json:"currentBackendIndex"`
	TrafficEnabled      bool   `json:"trafficEnabled"`
	Message             string `json:"message"`
}

func (c cluster) AsJSON() ClusterJSON {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return ClusterJSON{
		CurrentBackendIndex: uint(c.currentBackendIndex),

		// Traffic is enabled and disabled on all backends collectively
		// so we only need to read the state of one to get the state of
		// the system as a whole
		TrafficEnabled: c.backends.Any().TrafficEnabled(),

		Message: c.message,
	}
}

func (c *cluster) EnableTraffic() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.logger.Info("Enabling traffic for cluster")

	c.message = ""

	for backend := range c.backends.All() {
		backend.EnableTraffic()
	}
}

func (c *cluster) DisableTraffic(message string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.logger.Info("Disabling traffic for cluster", lager.Data{"message": message})

	c.message = message

	for backend := range c.backends.All() {
		backend.DisableTraffic()
	}
}
