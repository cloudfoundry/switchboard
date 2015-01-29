package domain

import (
	"errors"
	"fmt"
	"net"
	"net/http"
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

type Cluster interface {
	Monitor() chan<- interface{}
	RouteToBackend(clientConn net.Conn) error
}

type cluster struct {
	backends            Backends
	currentBackendIndex int
	logger              lager.Logger
	healthcheckTimeout  time.Duration
}

func NewCluster(backends Backends, healthcheckTimeout time.Duration, logger lager.Logger) Cluster {
	return cluster{
		backends:            backends,
		currentBackendIndex: 0,
		logger:              logger,
		healthcheckTimeout:  healthcheckTimeout,
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
		errorCount := uint64(0)
		logBadFrequency := uint64(5)
		for {
			select {
			case <-time.Tick(c.healthcheckTimeout / 5):
				url := backend.HealthcheckUrl()
				resp, err := client.Get(url)
				if err != nil {
					c.backends.SetUnhealthy(backend)

					errorCount++
					shouldLogError := errorCount%logBadFrequency == 0
					if shouldLogError {
						c.logger.Error(
							fmt.Sprintf("Error dialing healthchecker; total failures: %d", errorCount),
							err,
							lager.Data{
								"backend":       backend.AsJSON(),
								"endpoint":      url,
								"totalFailures": errorCount,
							},
						)
					}
				} else {
					resp.Body.Close()

					if resp.StatusCode == http.StatusOK {
						c.logger.Debug("Healthcheck succeeded", lager.Data{"endpoint": url})
						c.backends.SetHealthy(backend)
					} else {
						c.backends.SetUnhealthy(backend)

						errorCount++
						shouldLogError := errorCount%logBadFrequency == 0
						if shouldLogError {
							c.logger.Error(
								fmt.Sprintf("Healthcheck exit code: %d; total failures: %d", resp.StatusCode, errorCount),
								fmt.Errorf("Non-200 exit code from healthcheck"),
								lager.Data{
									"backend":       backend.AsJSON(),
									"endpoint":      url,
									"status_code":   resp.StatusCode,
									"totalFailures": errorCount,
								},
							)
						}
					}
				}
			case <-stopChan:
				return
			}
		}
	}()
}
