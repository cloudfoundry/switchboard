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
}

func NewCluster(backendIPs []string, backendPorts []uint, healthcheckPorts []uint, healthcheckTimeout time.Duration, logger lager.Logger) Cluster {
	healthchecks := newHealthchecks(backendIPs, healthcheckPorts, healthcheckTimeout, logger)
	backendSlice := make([]Backend, len(backendIPs))
	for i, ip := range backendIPs {
		backendSlice[i] = NewBackend(fmt.Sprintf("Backend-%d", i), ip, backendPorts[i], healthchecks[i])
	}
	return cluster{
		backends:            backendSlice,
		currentBackendIndex: 0,
		logger:              logger,
	}
}

func newHealthchecks(backendIPs []string, healthcheckPorts []uint, timeout time.Duration, logger lager.Logger) []Healthcheck {
	healthchecks := make([]Healthcheck, len(backendIPs))
	for i, ip := range backendIPs {
		healthchecks[i] = NewHttpHealthCheck(
			ip,
			healthcheckPorts[i],
			timeout,
			logger)
	}
	return healthchecks
}

func (c cluster) StartHealthchecks() {
	for _, backend := range c.backends {
		backend.StartHealthcheck()
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
