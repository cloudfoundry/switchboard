package monitor

import (
	"fmt"
	"net/http"
	"reflect"
	"time"

	"sync"

	"math"

	"encoding/json"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-incubator/galera-healthcheck/api"
	"github.com/cloudfoundry-incubator/switchboard/domain"
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

type BackendStatus struct {
	Index    int
	Healthy  bool
	Counters *DecisionCounters
}

type Cluster struct {
	backends                 []*domain.Backend
	logger                   lager.Logger
	healthcheckTimeout       time.Duration
	arpManager               ArpManager
	activeBackendSubscribers []chan<- *domain.Backend
}

func NewCluster(
	backends []*domain.Backend,
	healthcheckTimeout time.Duration,
	logger lager.Logger,
	arpManager ArpManager,
	activeBackendSubscribers []chan<- *domain.Backend,
) *Cluster {
	return &Cluster{
		backends:                 backends,
		logger:                   logger,
		healthcheckTimeout:       healthcheckTimeout,
		arpManager:               arpManager,
		activeBackendSubscribers: activeBackendSubscribers,
	}
}

func (c *Cluster) Monitor(stopChan <-chan interface{}) {
	client := UrlGetterProvider(c.healthcheckTimeout)

	backendHealthMap := make(map[*domain.Backend]*BackendStatus)

	for i, backend := range c.backends {
		backendHealthMap[backend] = &BackendStatus{
			Index:    i,
			Counters: c.SetupCounters(),
		}
	}

	go func() {
		var activeBackend *domain.Backend

		for {
			select {
			case <-time.After(c.healthcheckTimeout / 5):
				var wg sync.WaitGroup

				for backend, healthStatus := range backendHealthMap {
					wg.Add(1)
					go func(backend *domain.Backend, healthStatus *BackendStatus) {
						defer wg.Done()
						c.QueryBackendHealth(backend, healthStatus, client)
					}(backend, healthStatus)
				}

				wg.Wait()

				newActiveBackend := ChooseActiveBackend(backendHealthMap)

				if newActiveBackend != activeBackend {
					if newActiveBackend != nil {
						c.logger.Info("New active backend", lager.Data{"backend": newActiveBackend.AsJSON()})
					}

					activeBackend = newActiveBackend
					for _, s := range c.activeBackendSubscribers {
						s <- activeBackend
					}
				}

			case <-stopChan:
				return
			}

		}

	}()

}

func (c *Cluster) SetupCounters() *DecisionCounters {
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

func ChooseActiveBackend(backendHealths map[*domain.Backend]*BackendStatus) *domain.Backend {
	var lowestIndexedHealthyDomain *domain.Backend
	lowestHealthyIndex := math.MaxUint32

	for backend, backendStatus := range backendHealths {
		if !backendStatus.Healthy || lowestHealthyIndex <= backendStatus.Index {
			continue
		}

		lowestHealthyIndex = backendStatus.Index
		lowestIndexedHealthyDomain = backend
	}

	return lowestIndexedHealthyDomain
}

func (c *Cluster) determineStateFromBackend(backend *domain.Backend, client UrlGetter, shouldLog bool) (bool, *int) {
	j := backend.AsJSON()

	url := fmt.Sprintf("http://%s:%d/api/v1/status", j.Host, j.StatusPort)
	resp, err := client.Get(url)

	/////////////////
	// Determine health from either the v1 status endpoint
	// or fallback to the v0 status endpoint
	var healthy bool
	var index *int

	if err == nil {
		if resp.StatusCode == http.StatusOK {
			var v1StatusResponse api.V1StatusResponse

			_ = json.NewDecoder(resp.Body).Decode(&v1StatusResponse)

			healthy = v1StatusResponse.Healthy
			indexVal := int(v1StatusResponse.WsrepLocalIndex)
			index = &indexVal
		} else if resp.StatusCode == http.StatusNotFound {
			url = backend.HealthcheckUrl()
			resp, err = client.Get(url)

			healthy = (err == nil && resp.StatusCode == http.StatusOK)
		}
	}

	if shouldLog {
		if !healthy && err == nil {
			c.logger.Error(
				"Healthcheck failed on backend",
				fmt.Errorf("Backend reported as unhealthy"),
				lager.Data{
					"backend":  backend.AsJSON(),
					"endpoint": url,
					"resp":     fmt.Sprintf("%#v", resp),
				},
			)
		}

		if err != nil {
			c.logger.Error(
				"Healthcheck failed on backend",
				fmt.Errorf("Error during healthcheck http get"),
				lager.Data{
					"backend":  backend.AsJSON(),
					"endpoint": url,
					"resp":     fmt.Sprintf("%#v", resp),
					"err":      err,
				},
			)
		}

		if healthy {
			c.logger.Debug("Healthcheck succeeded", lager.Data{"endpoint": url})
		}
	}

	return healthy, index
}

func (c *Cluster) QueryBackendHealth(backend *domain.Backend, healthMonitor *BackendStatus, client UrlGetter) {
	healthMonitor.Counters.IncrementCount("dial")
	shouldLog := healthMonitor.Counters.Should("log")

	healthy, index := c.determineStateFromBackend(backend, client, shouldLog)

	if index != nil {
		healthMonitor.Index = *index
	}

	if healthy {
		backend.SetHealthy()
		healthMonitor.Healthy = true
		healthMonitor.Counters.ResetCount("consecutiveUnhealthyChecks")
	} else {
		backend.SetUnhealthy()
		healthMonitor.Healthy = false
		healthMonitor.Counters.IncrementCount("consecutiveUnhealthyChecks")

	}

	if healthMonitor.Counters.Should("clearArp") {
		backendHost := backend.AsJSON().Host

		if c.arpManager.IsCached(backendHost) {
			err := c.arpManager.ClearCache(backendHost)
			if err != nil {
				c.logger.Error("Failed to clear arp cache", err)
			}
		}
	}
}
