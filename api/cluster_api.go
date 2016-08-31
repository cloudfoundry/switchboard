package api

import (
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/switchboard/domain"
	"github.com/pivotal-golang/lager"
)

type ClusterAPI struct {
	mutex       sync.RWMutex
	backends    Backends
	logger      lager.Logger
	message     string
	lastUpdated time.Time
}

//go:generate counterfeiter . Backends
type Backends interface {
	All() <-chan domain.Backend
}

func NewClusterAPI(backends Backends, logger lager.Logger) *ClusterAPI {
	return &ClusterAPI{
		backends: backends,
		logger:   logger,
	}
}

func anyBackend(backends Backends) domain.Backend {
	for backend := range backends.All() {
		return backend
	}
	return nil
}

func (c *ClusterAPI) AsJSON() ClusterJSON {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return ClusterJSON{
		TrafficEnabled: true,

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

}

func (c *ClusterAPI) DisableTraffic(message string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.logger.Info("Disabling traffic for cluster", lager.Data{"message": message})

	c.message = message
	c.lastUpdated = time.Now()
}

type ClusterJSON struct {
	TrafficEnabled bool      `json:"trafficEnabled"`
	Message        string    `json:"message"`
	LastUpdated    time.Time `json:"lastUpdated"`
}
