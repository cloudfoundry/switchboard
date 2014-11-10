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

type backends []Backend

func NewBackends(backendIPs []string, backendPorts []uint, healthcheckPorts []uint, healthcheckTimeout time.Duration, logger lager.Logger) Backends {
	healthchecks := newHealthchecks(backendIPs, healthcheckPorts, healthcheckTimeout, logger)
	backendSlice := make([]Backend, len(backendIPs))
	for i, ip := range backendIPs {
		backendSlice[i] = NewBackend(fmt.Sprintf("Backend-%d", i), ip, backendPorts[i], healthchecks[i])
	}
	return backends(backendSlice)
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

func (backends backends) StartHealthchecks() {
	for _, backend := range backends {
		backend.StartHealthcheck()
	}
}

func (backends backends) CurrentBackend() Backend {
	currentBackendIndex := 0
	return backends[currentBackendIndex]
}
