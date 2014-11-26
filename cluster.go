package switchboard

import (
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/pivotal-golang/lager"
)

type Cluster interface {
	Monitor()
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

func (c cluster) Monitor() {
	for backend := range c.backends.All() {
		go c.monitorHealth(backend)
	}
}

func (c cluster) RouteToBackend(clientConn net.Conn) error {
	activeBackend := c.backends.Active()
	if activeBackend == nil {
		return errors.New("No active Backend")
	}
	return activeBackend.Bridge(clientConn)
}

func (c cluster) monitorHealth(backend Backend) {
	for _ = range time.Tick(c.healthcheckTimeout / 5) {
		url := backend.HealthcheckUrl()
		client := http.Client{
			Timeout: c.healthcheckTimeout,
		}

		resp, err := client.Get(url)
		if err != nil {
			c.logger.Error("Error dialing healthchecker", err, lager.Data{"endpoint": url})
			c.backends.SetUnhealthy(backend)
		} else {
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				c.logger.Debug("Healthcheck succeeded", lager.Data{"endpoint": url})
				c.backends.SetHealthy(backend)
			} else {
				c.logger.Debug("Non-200 exit code from healthcheck", lager.Data{"status_code": resp.StatusCode, "endpoint": url})
				c.backends.SetUnhealthy(backend)
			}
		}
	}
}
