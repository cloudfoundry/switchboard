package domain

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"time"

	"github.com/pivotal-golang/lager"
	"github.com/cloudfoundry-incubator/switchboard/models"
)

//go:generate counterfeiter . UrlGetter
type UrlGetter interface {
	Get(url string) (*http.Response, error)
}

func HttpUrlGetterProvider(healthcheckTimeout time.Duration) UrlGetter {
	return &http.Client{
		Timeout: healthcheckTimeout,
	}
}

var UrlGetterProvider = HttpUrlGetterProvider

type Cluster struct {
	backends            models.Backends
	currentBackendIndex int
	logger              lager.Logger
	healthcheckTimeout  time.Duration
	arpManager          models.ArpManager
}

func NewCluster(backends models.Backends, healthcheckTimeout time.Duration, logger lager.Logger, arpManager models.ArpManager) *Cluster {
	return &Cluster{
		backends:            backends,
		currentBackendIndex: 0,
		logger:              logger,
		healthcheckTimeout:  healthcheckTimeout,
		arpManager:          arpManager,
	}
}

func (c Cluster) Monitor() chan<- interface{} {
	client := UrlGetterProvider(c.healthcheckTimeout)
	stopChan := make(chan interface{})
	for backend := range c.backends.All() {
		c.monitorHealth(backend, client, stopChan)
	}
	return stopChan
}

func (c Cluster) RouteToBackend(clientConn net.Conn) error {
	activeBackend := c.backends.Active()
	if activeBackend == nil {
		return errors.New("No active Backend")
	}
	return activeBackend.Bridge(clientConn)
}

func (c Cluster) monitorHealth(backend models.Backend, client UrlGetter, stopChan <-chan interface{}) {
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

func (c Cluster) setupCounters() *DecisionCounters {
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

func (c Cluster) dialHealthcheck(backend models.Backend, client UrlGetter, counters *DecisionCounters) {

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

	// FIXME: This is gross.
	if counters.Should("clearArp") {
		backend, ok := backend.AsJSON().(BackendJSON)
		if !ok {
			return
		}

		backendHost := backend.Host

		if c.arpManager.IsCached(backendHost) {
			err = c.arpManager.ClearCache(backendHost)
			if err != nil {
				c.logger.Error("Failed to clear arp cache", err)
			}
		}
	}
}
