package api

import (
	"sync"
	"time"

	"github.com/pivotal-golang/lager"
)

type ClusterAPI struct {
	mutex              sync.RWMutex
	logger             lager.Logger
	message            string
	lastUpdated        time.Time
	trafficEnabled     bool
	trafficEnabledChan chan<- bool
}

func NewClusterAPI(trafficEnabledChan chan<- bool, logger lager.Logger) *ClusterAPI {
	return &ClusterAPI{
		logger:             logger,
		trafficEnabled:     true,
		trafficEnabledChan: trafficEnabledChan,
	}
}

func (c *ClusterAPI) AsJSON() ClusterJSON {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return ClusterJSON{
		TrafficEnabled: c.trafficEnabled,
		Message:        c.message,
		LastUpdated:    c.lastUpdated,
	}
}

func (c *ClusterAPI) EnableTraffic(message string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.logger.Info("Enabling traffic for cluster", lager.Data{"message": message})

	c.message = message
	c.lastUpdated = time.Now()
	c.trafficEnabled = true

	c.trafficEnabledChan <- c.trafficEnabled
}

func (c *ClusterAPI) DisableTraffic(message string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.logger.Info("Disabling traffic for cluster", lager.Data{"message": message})

	c.message = message
	c.lastUpdated = time.Now()
	c.trafficEnabled = false

	c.trafficEnabledChan <- c.trafficEnabled
}

type ClusterJSON struct {
	TrafficEnabled bool      `json:"trafficEnabled"`
	Message        string    `json:"message"`
	LastUpdated    time.Time `json:"lastUpdated"`
}
