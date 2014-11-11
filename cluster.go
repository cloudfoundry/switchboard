package switchboard

import (
	"fmt"
	"net"
	"time"

	"github.com/pivotal-golang/lager"
)

type Cluster interface {
	StartHealthchecks()
	RouteToBackend(clientConn net.Conn)
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
		backends[i] = NewBackend(fmt.Sprintf("Backend-%d", i), ip, backendPorts[i], healthcheckPorts[i])
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
		healthcheck := NewHttpHealthCheck(c.healthcheckTimeout, c.logger)
		healthcheck.Start(backend)
	}
}

func (c cluster) RouteToBackend(clientConn net.Conn) {
	backend := c.currentBackend()
	backendConn, err := backend.Dial()
	if err != nil {
		c.logger.Error("Error connection to backend.", err)
		return
	}

	bridge := NewConnectionBridge(clientConn, backendConn, c.logger)
	backend.AddBridge(bridge)

	go func() {
		bridge.Connect()
		backend.RemoveBridge(bridge)
	}()
}

func (c cluster) currentBackend() Backend {
	return c.backends[c.currentBackendIndex]
}
