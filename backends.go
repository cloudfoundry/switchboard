package switchboard

import (
	"fmt"
	"time"

	"github.com/pivotal-golang/lager"
)

type Backends interface {
	StartHealthchecks()
	CurrentBackend() Backend
}

type backends struct {
	backends            []Backend
	currentBackendIndex int
}

func NewBackends(backendIPs []string, backendPorts []uint, healthcheckPorts []uint, healthcheckTimeout time.Duration, logger lager.Logger) Backends {
	healthchecks := newHealthchecks(backendIPs, healthcheckPorts, healthcheckTimeout, logger)
	backendSlice := make([]Backend, len(backendIPs))
	for i, ip := range backendIPs {
		backendSlice[i] = NewBackend(fmt.Sprintf("Backend-%d", i), ip, backendPorts[i], healthchecks[i])
	}
	return backends{
		backends:            backendSlice,
		currentBackendIndex: 0,
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

func (b backends) StartHealthchecks() {
	for _, backend := range b.backends {
		backend.StartHealthcheck()
	}
}

func (b backends) CurrentBackend() Backend {
	return b.backends[b.currentBackendIndex]
}
