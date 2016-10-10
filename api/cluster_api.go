package api

import (
	"sync"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-incubator/switchboard/domain"
)

type ClusterAPI struct {
	mutex              sync.RWMutex
	logger             lager.Logger
	message            string
	lastUpdated        time.Time
	trafficEnabled     bool
	trafficEnabledChan chan<- bool
	activeBackendChan  <-chan *domain.Backend
	activeBackend      *BackendJSON
}

func NewClusterAPI(
	trafficEnabledChan chan<- bool,
	activeBackendChan <-chan *domain.Backend,
	logger lager.Logger) *ClusterAPI {
	return &ClusterAPI{
		logger:             logger,
		trafficEnabled:     true,
		trafficEnabledChan: trafficEnabledChan,
		activeBackendChan:  activeBackendChan,
	}
}

func (c *ClusterAPI) ListenForActiveBackend() {
	for b := range c.activeBackendChan {
		c.mutex.Lock()

		if b == nil {
			c.activeBackend = nil
		} else {
			j := b.AsJSON()
			c.activeBackend = &BackendJSON{
				Host: j.Host,
				Port: j.Port,
				Name: j.Name,
			}
		}
		c.mutex.Unlock()
	}
}

func (c *ClusterAPI) AsJSON() ClusterJSON {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return ClusterJSON{
		TrafficEnabled: c.trafficEnabled,
		Message:        c.message,
		LastUpdated:    c.lastUpdated,
		ActiveBackend:  c.activeBackend,
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
	ActiveBackend  *BackendJSON `json:"activeBackend"`
	TrafficEnabled bool         `json:"trafficEnabled"`
	Message        string       `json:"message"`
	LastUpdated    time.Time    `json:"lastUpdated"`
}

type BackendJSON struct {
	Host string `json:"host"`
	Port uint   `json:"port"`
	Name string `json:"name"`
}
