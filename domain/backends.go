package domain

import (
	"sync"

	"github.com/cloudfoundry-incubator/switchboard/config"
	"github.com/cloudfoundry-incubator/switchboard/models"
	"github.com/pivotal-golang/lager"
)

var BackendProvider = func(name string,
	host string,
	port uint,
	statusPort uint,
	statusEndpoint string,
	logger lager.Logger) models.Backend {
	return NewBackend(name, host, port, statusPort, statusEndpoint, logger)
}

type Backends struct {
	mutex  sync.RWMutex
	all    map[models.Backend]bool
	active models.Backend
	Logger lager.Logger
}

func NewBackends(backendConfigs []config.Backend, logger lager.Logger) *Backends {
	b := &Backends{
		Logger: logger,
		all:    make(map[models.Backend]bool),
	}

	for _, bc := range backendConfigs {
		backend := BackendProvider(
			bc.Name,
			bc.Host,
			bc.Port,
			bc.StatusPort,
			bc.StatusEndpoint,
			logger,
		)

		b.all[backend] = true
	}

	b.active = b.unsafeNextHealthy()

	return b
}

// Maintains a lock while iterating over all backends
func (b *Backends) All() <-chan models.Backend {
	b.mutex.RLock()

	ch := make(chan models.Backend, len(b.all))
	go func() {
		defer b.mutex.RUnlock()

		for backend := range b.all {
			ch <- backend
		}
		close(ch)
	}()

	return ch
}

// Maintains a lock while iterating over healthy backends
func (b *Backends) Healthy() <-chan models.Backend {
	b.mutex.RLock()

	c := make(chan models.Backend, len(b.all))
	go func() {
		defer b.mutex.RUnlock()

		for backend, healthy := range b.all {
			if healthy {
				c <- backend
			}
		}

		close(c)
	}()

	return c
}

func (b *Backends) Active() models.Backend {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	return b.active
}

func (b *Backends) SetHealthy(backend models.Backend) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	previouslyHeathly := b.all[backend]
	if !previouslyHeathly {
		b.Logger.Info("Previously unhealthy backend became healthy.", lager.Data{"backend": backend.AsJSON()})
	}

	b.all[backend] = true
	if b.active == nil {
		b.unsafeSetActive(backend)
	}
}

func (b *Backends) unsafeSetActive(backend models.Backend) {
	b.active = backend

	if b.active == nil {
		b.Logger.Info("No active backends.")
	} else {
		b.Logger.Info("New active backend", lager.Data{"backend": b.active.AsJSON()})
	}
}

func (b *Backends) SetUnhealthy(backend models.Backend) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	previouslyHeathly := b.all[backend]
	if previouslyHeathly {
		b.Logger.Info("Previously healthy backend became unhealthy.", lager.Data{"backend": backend.AsJSON()})
	}

	b.all[backend] = false
	if b.active == backend {
		backend.SeverConnections()
		nextHealthyBackend := b.unsafeNextHealthy()
		b.unsafeSetActive(nextHealthyBackend)
	}
}

func (b *Backends) AsJSON() interface{} {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	backendsJSON := []BackendJSON{}
	for backend, healthy := range b.all {
		backendJSON, ok := backend.AsJSON().(BackendJSON)
		if !ok {
			return nil
		}

		backendJSON.Healthy = healthy
		backendJSON.Active = (b.active == backend)
		backendsJSON = append(backendsJSON, backendJSON)
	}

	return backendsJSON
}

func (b *Backends) unsafeNextHealthy() models.Backend {
	for backend, healthy := range b.all {
		if healthy {
			return backend
		}
	}
	return nil
}
