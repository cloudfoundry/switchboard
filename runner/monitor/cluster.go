package monitor

import (
	"fmt"
	"net/http"
	"reflect"
	"time"

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

type backendMonitor struct {
	backend  *domain.Backend
	healthy  bool
	counters *DecisionCounters
}

type Cluster struct {
	backends           []*domain.Backend
	logger             lager.Logger
	healthcheckTimeout time.Duration
	arpManager         ArpManager
	activeBackendChan  chan<- domain.IBackend
}

func NewCluster(backends []*domain.Backend, healthcheckTimeout time.Duration, logger lager.Logger, arpManager ArpManager, activeBackendChan chan<- domain.IBackend) *Cluster {
	return &Cluster{
		backends:           backends,
		logger:             logger,
		healthcheckTimeout: healthcheckTimeout,
		arpManager:         arpManager,
		activeBackendChan:  activeBackendChan,
	}
}

func (c *Cluster) Monitor(stopChan <-chan interface{}) {
	client := UrlGetterProvider(c.healthcheckTimeout)

	var backendHealth []*backendMonitor

	for _, backend := range c.backends {
		backendHealth = append(backendHealth,
			&backendMonitor{
				backend:  backend,
				counters: c.setupCounters(),
			})
	}

	go func() {
		var activeBackend domain.IBackend
		for {

			select {
			case <-time.After(c.healthcheckTimeout / 5):
				for _, healthMonitor := range backendHealth {
					backend := healthMonitor.backend

					healthMonitor.counters.IncrementCount("dial")
					shouldLog := healthMonitor.counters.Should("log")

					url := backend.HealthcheckUrl()
					resp, err := client.Get(url)

					if err == nil && resp.StatusCode == http.StatusOK {
						backend.SetHealthy()
						healthMonitor.healthy = true
						healthMonitor.counters.ResetCount("consecutiveUnhealthyChecks")

						if shouldLog {
							c.logger.Debug("Healthcheck succeeded", lager.Data{"endpoint": url})
						}
					} else {
						backend.SetUnhealthy()
						healthMonitor.healthy = false
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
				}

				var anyHealthy bool
				for _, healthMonitor := range backendHealth {
					backend := healthMonitor.backend

					if healthMonitor.healthy {
						anyHealthy = true
					}

					if healthMonitor.healthy && activeBackend == backend {
						break
					}

					if healthMonitor.healthy && activeBackend != backend {
						c.logger.Info("New active backend", lager.Data{"backend": backend.AsJSON()})
						activeBackend = backend
						c.activeBackendChan <- activeBackend
						break
					}
				}

				if !anyHealthy {
					c.logger.Info("No active backends.")
					c.activeBackendChan <- nil
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
