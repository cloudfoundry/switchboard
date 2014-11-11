package switchboard

import (
	"net"
	"time"

	"github.com/pivotal-golang/lager"
)

type Cluster interface {
	StartHealthchecks()
	RouteToBackend(clientConn net.Conn) error
}

type cluster struct {
	backends            []Backend
	currentBackendIndex int
	logger              lager.Logger
	healthcheckTimeout  time.Duration
}

func NewCluster(backendIPs []string, backendPorts []uint, healthcheckPorts []uint, healthcheckTimeout time.Duration, logger lager.Logger) Cluster {
	backends := make([]Backend, len(backendIPs))
	for i, ip := range backendIPs {
		backends[i] = NewBackend(
			ip,
			backendPorts[i],
			healthcheckPorts[i],
			logger,
		)
	}
	return cluster{
		backends:            backends,
		currentBackendIndex: 0,
		logger:              logger,
		healthcheckTimeout:  healthcheckTimeout,
	}
}

func (c cluster) StartHealthchecks() {
	for _, backend := range c.backends {
		healthcheck := NewHealthcheck(c.healthcheckTimeout, c.logger)
		healthcheck.Start(backend)
	}
}

func (c cluster) RouteToBackend(clientConn net.Conn) error {
	backend := c.backends[c.currentBackendIndex]
	return backend.Bridge(clientConn)
}
