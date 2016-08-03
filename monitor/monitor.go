package monitor

import (
	"fmt"
	"net/http"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-incubator/switchboard/domain"
	"github.com/cloudfoundry-incubator/switchboard/statemachine"
)

type StateMachine interface {
	BecomesHealthy()
	BecomesUnhealthy()
}

//go:generate counterfeiter . UrlGetter
type URLGetter interface {
	Get(url string) (*http.Response, error)
}

func HTTPURLGetterProvider(healthcheckTimeout time.Duration) URLGetter {
	return &http.Client{
		Timeout: healthcheckTimeout,
	}
}

var URLGetterProvider = HTTPURLGetterProvider

type ClusterMonitor struct {
	Backends           domain.Backends
	HealthcheckTimeout time.Time
	URLGetter          URLGetter
	Logger             lager.Logger
	StateMachine       StateMachine
}

func NewClusterMonitor(backends domain.Backends, healthcheckTimeout time.Time, logger lager.Logger) *ClusterMonitor {
	return &ClusterMonitor{
		Backends:           backends,
		HealthcheckTimeout: healthcheckTimeout,
		URLGetter:          URLGetterProvider(healthcheckTimeout),
		Logger:             logger,
		StateMachine: &statemachine.StatefulStateMachine{
			Logger: lager.NewLogger("StateMachine"),
			OnBecomesHealthy: func(backend domain.Backend) {
				backends.SetHealthy(backend)


			},
			OnBecomesUnhealthy: func(backend domain.Backend) {
				backends.SetUnhealthy(backend)
			},
		},
	}
}

func (c *ClusterMonitor) Monitor(stopChan chan<- interface{}) {
	for backend := range c.Backends.All() {
		go c.monitorHealth(backend, c.URLGetter, stopChan)
	}
}

func (c *ClusterMonitor) monitorHealth(backend domain.Backend, client URLGetter, stopChan <-chan interface{}) {
	consecutiveUnhealthyChecks := 0
	numLogs := 0

	logFreq := 5

	for {
		select {
		case <-time.After(c.HealthcheckTimeout / 5):
			numLogs++
			shouldLog := numLogs%logFreq == 0

			url := backend.HealthcheckUrl()
			resp, err := client.Get(url)

			if err == nil && resp.StatusCode == http.StatusOK {
				c.Backends.SetHealthy(backend)
				consecutiveUnhealthyChecks = 0

				if shouldLog {
					c.Logger.Debug("Healthcheck succeeded", lager.Data{"endpoint": url})
				}
			} else {
				c.Backends.SetUnhealthy(backend)
				consecutiveUnhealthyChecks++

				if shouldLog {
					c.Logger.Error(
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
		case <-stopChan:
			return
		}
	}
}
