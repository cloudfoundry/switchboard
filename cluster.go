package switchboard

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/pivotal-golang/lager"
)

type Cluster interface {
	StartHealthchecks()
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

func (c cluster) StartHealthchecks() {
	for backend := range c.backends.All() {
		healthcheck := NewHealthcheck(c.healthcheckTimeout, c.logger)
		healthyChan, unhealthyChan := healthcheck.Start(backend)
		c.watchForUnhealthy(healthyChan, unhealthyChan)
	}
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
		fmt.Println("Received unhealthy state")
		oldActive := c.backends.Active()
		c.backends.SetUnhealthy(backend)
		fmt.Printf("Backends.SetUnhealthy to backend at %s\n", backend.HealthcheckUrl())

		if oldActive == backend {
			fmt.Println("Sever Connections if active")
			backend.SeverConnections()
		}

		c.watchForHealthy(healthyChan, unhealthyChan)
	}()
}

// Watches for an unhealthy backend to become healthy again
func (c cluster) watchForHealthy(healthyChan <-chan Backend, unhealthyChan <-chan Backend) {
	go func() {
		backend := <-healthyChan
		fmt.Println("Received healthy state")
		c.backends.SetHealthy(backend)
		fmt.Printf("Backends.SetHealthy to backend at %s\n", backend.HealthcheckUrl())
		c.watchForUnhealthy(healthyChan, unhealthyChan)
	}()
}
