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
		healthcheck.Start(backend)
	}
}

func (c cluster) RouteToBackend(clientConn net.Conn) error {
	return c.backends.Active().Bridge(clientConn)
}
