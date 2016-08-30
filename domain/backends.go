package domain

import (
	"sync"

	"github.com/cloudfoundry-incubator/switchboard/config"
	"github.com/pivotal-golang/lager"
)

var BackendProvider = NewBackend

type BackendsRepository struct {
	mutex  sync.RWMutex
	all    map[Backend]bool
	active Backend
	logger lager.Logger
}

func NewBackends(backendConfigs []config.Backend, logger lager.Logger) *BackendsRepository {
	b := &BackendsRepository{
		logger: logger,
		all:    make(map[Backend]bool),
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
func (b *BackendsRepository) All() <-chan Backend {
	b.mutex.RLock()

	ch := make(chan Backend, len(b.all))
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
// used only in backends test
func (b *BackendsRepository) Healthy() <-chan Backend {
	b.mutex.RLock()

	c := make(chan Backend, len(b.all))
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

func (b *BackendsRepository) Active() Backend {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	return b.active
}

func (b *BackendsRepository) SetHealthy(backend Backend) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	previouslyHealthy := b.all[backend]
	if !previouslyHealthy {
		b.logger.Info("Previously unhealthy backend became healthy.", lager.Data{"backend": backend.AsJSON()})
	}

	b.all[backend] = true
	if b.active == nil {
		b.unsafeSetActive(backend)
	}
}

func (b *BackendsRepository) unsafeSetActive(backend Backend) {
	b.active = backend

	if b.active == nil {
		b.logger.Info("No active backends.")
	} else {
		b.logger.Info("New active backend", lager.Data{"backend": b.active.AsJSON()})
	}
}

func (b *BackendsRepository) SetUnhealthy(backend Backend) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	previouslyHealthy := b.all[backend]
	if previouslyHealthy {
		b.logger.Info("Previously healthy backend became unhealthy.", lager.Data{"backend": backend.AsJSON()})
	}

	b.all[backend] = false
	if b.active == backend {
		backend.SeverConnections()
		nextHealthyBackend := b.unsafeNextHealthy()
		b.unsafeSetActive(nextHealthyBackend)
	}
}

func (b *BackendsRepository) AsJSON() []BackendJSON {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	backendsJSON := []BackendJSON{}
	for backend, healthy := range b.all {
		backendJSON := backend.AsJSON()
		backendJSON.Healthy = healthy
		backendJSON.Active = (b.active == backend)
		backendsJSON = append(backendsJSON, backendJSON)
	}

	return backendsJSON
}

func (b *BackendsRepository) unsafeNextHealthy() Backend {
	for backend, healthy := range b.all {
		if healthy {
			return backend
		}
	}
	return nil
}
