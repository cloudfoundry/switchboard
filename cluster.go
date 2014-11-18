package switchboard

import (
	"errors"
	"net"
	"time"

	"github.com/pivotal-golang/lager"
)

type Cluster interface {
	Start() (<-chan struct{}, <-chan struct{})
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

func (c cluster) Start() (<-chan struct{}, <-chan struct{}) {
	c.logger.Info("Starting cluster ...")
	upChan := make(chan struct{})
	downChan := make(chan struct{})

	for backend := range c.backends.All() {
		healthcheck := NewHealthcheck(c.healthcheckTimeout, c.logger)
		healthyChan, unhealthyChan := healthcheck.Start(backend)
		c.watchForUnhealthy(healthyChan, unhealthyChan)
	}

	go func() {
		activeChan, inactiveChan := c.backends.ActivityChannels()
		for {
			select {
			case <-activeChan:
				c.logger.Info("Backends active. Cluster is up again.")
				upChan <- struct{}{}
			case <-inactiveChan:
				c.logger.Info("Backends inactive. Cluster is down.")
				downChan <- struct{}{}
			}
		}
	}()

	return upChan, downChan
}

func (c cluster) RouteToBackend(clientConn net.Conn) error {
	activeBackend := c.backends.Active()
	if activeBackend == nil {
		return errors.New("No active Backend")
	}
	return activeBackend.Bridge(clientConn)
}

// Watches for a healthy backend to become unhealthy
func (c cluster) watchForUnhealthy(healthyChan <-chan Backend, unhealthyChan <-chan Backend) {
	go func() {
		backend := <-unhealthyChan
		c.logger.Info("Healthcheck reported unhealthy backend.")
		oldActive := c.backends.Active()
		c.backends.SetUnhealthy(backend)

		if oldActive == backend {
			c.logger.Info("Unhealthy backend used to be the active one. Severing existing connections!")
			backend.SeverConnections()
		}

		c.waitForHealthy(healthyChan, unhealthyChan)
	}()
}

// Watches for an unhealthy backend to become healthy again
func (c cluster) waitForHealthy(healthyChan <-chan Backend, unhealthyChan <-chan Backend) {
	go func() {
		backend := <-healthyChan
		c.logger.Info("Healthcheck reported healthy backend")
		c.backends.SetHealthy(backend)
		c.watchForUnhealthy(healthyChan, unhealthyChan)
	}()
}
