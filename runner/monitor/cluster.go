package monitor

import (
	"fmt"
	"net/http"
	"reflect"
	"time"

	"sync"

	"math"

	"github.com/cloudfoundry-incubator/switchboard/domain"
	"github.com/pivotal-golang/lager"
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
	counters *DecisionCounters
}

type Cluster struct {
	backends                 []*domain.Backend
	logger                   lager.Logger
	healthcheckTimeout       time.Duration
	arpManager               ArpManager
	activeBackendSubscribers []chan<- *domain.Backend
}

func NewCluster(backends []*domain.Backend, healthcheckTimeout time.Duration, logger lager.Logger, arpManager ArpManager, activeBackendSubscribers []chan<- *domain.Backend) *Cluster {
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
			counters: c.setupCounters(),
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
					go func(backend *domain.Backend, healthMonitor *BackendStatus) {
						defer wg.Done()

						healthMonitor.counters.IncrementCount("dial")
						shouldLog := healthMonitor.counters.Should("log")

						url := backend.HealthcheckUrl()
						resp, err := client.Get(url)

						if err == nil && resp.StatusCode == http.StatusOK {
							backend.SetHealthy()
							healthMonitor.Healthy = true
							healthMonitor.counters.ResetCount("consecutiveUnhealthyChecks")

							if shouldLog {
								c.logger.Debug("Healthcheck succeeded", lager.Data{"endpoint": url})
							}
						} else {
							backend.SetUnhealthy()
							healthMonitor.Healthy = false
							healthMonitor.counters.IncrementCount("consecutiveUnhealthyChecks")

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

						if healthMonitor.counters.Should("clearArp") {
							backendHost := backend.AsJSON().Host

							if c.arpManager.IsCached(backendHost) {
								err = c.arpManager.ClearCache(backendHost)
								if err != nil {
									c.logger.Error("Failed to clear arp cache", err)
								}
							}
						}
					}(backend, healthStatus)

				}

				wg.Wait()

				newActiveBackend := ChooseActiveBackend(backendHealthMap)

				if newActiveBackend != activeBackend {
					activeBackend = newActiveBackend
					for _, s := range c.activeBackendSubscribers {
						s <- activeBackend
					}
				}

				if newActiveBackend != nil {
					c.logger.Info("New active backend", lager.Data{"backend": newActiveBackend.AsJSON()})
				}

			case <-stopChan:
				return
			}

		}

	}()

}

func (c *Cluster) setupCounters() *DecisionCounters {
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
