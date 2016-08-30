package api

import (
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/switchboard/domain"
	"github.com/pivotal-golang/lager"
)

type ClusterAPI struct {
	mutex       sync.RWMutex
	backends    domain.Backends
	logger      lager.Logger
	message     string
	lastUpdated time.Time
}

func NewClusterAPI(backends domain.Backends, logger lager.Logger) *ClusterAPI {
	return &ClusterAPI{
		backends: backends,
		logger:   logger,
	}
}

func anyBackend(backends domain.Backends) domain.Backend {
	for backend := range backends.All() {
		return backend
	}
	return nil
}

func (c *ClusterAPI) AsJSON() ClusterJSON {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return ClusterJSON{
		// Traffic is enabled and disabled on all backends collectively
		// so we only need to read the state of one to get the state of
		// the system as a whole
		// Test the nillable
		TrafficEnabled: anyBackend(c.backends).TrafficEnabled(),

		Message:     c.message,
		LastUpdated: c.lastUpdated,
	}
}

func (c *ClusterAPI) EnableTraffic(message string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.logger.Info("Enabling traffic for cluster", lager.Data{"message": message})

	c.message = message
	c.lastUpdated = time.Now()

	for backend := range c.backends.All() {
		backend.EnableTraffic()
	}
}

func (c *ClusterAPI) DisableTraffic(message string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.logger.Info("Disabling traffic for cluster", lager.Data{"message": message})

	c.message = message
	c.lastUpdated = time.Now()

	for backend := range c.backends.All() {
		backend.DisableTraffic()
	}
}

type ClusterJSON struct {
	TrafficEnabled bool      `json:"trafficEnabled"`
	Message        string    `json:"message"`
	LastUpdated    time.Time `json:"lastUpdated"`
}
