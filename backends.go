package switchboard

import (
	"sync"

	"github.com/pivotal-golang/lager"
)

type Backends interface {
	All() <-chan Backend
	Any() Backend
	Active() Backend
	SetHealthy(backend Backend)
	SetUnhealthy(backend Backend)
	Healthy() <-chan Backend
	ActivityChannels() (<-chan struct{}, <-chan struct{})
}

type backends struct {
	mutex        sync.RWMutex
	all          map[Backend]bool
	active       Backend
	logger       lager.Logger
	activeChan   chan struct{}
	inactiveChan chan struct{}
}

func NewBackends(backendIPs []string, backendPorts []uint, healthcheckPorts []uint, logger lager.Logger) Backends {
	b := &backends{
		logger:       logger,
		all:          make(map[Backend]bool),
		activeChan:   make(chan struct{}),
		inactiveChan: make(chan struct{}, 1),
	}

	for i, ip := range backendIPs {
		backend := NewBackend(
			ip,
			backendPorts[i],
			healthcheckPorts[i],
			logger,
		)

		b.all[backend] = true
	}

	if len(b.all) > 0 {
		b.active = b.unsafeNextHealthy()
	} else {
		select {
		case b.inactiveChan <- struct{}{}:
		default:
		}
	}

	return b
}

func (b *backends) ActivityChannels() (<-chan struct{}, <-chan struct{}) {
	return b.activeChan, b.inactiveChan
}

// Maintains a lock while iterating over all backends
func (b *backends) All() <-chan Backend {
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
func (b *backends) Healthy() <-chan Backend {
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

func (b *backends) Any() Backend {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	for backend := range b.all {
		return backend
	}

	return nil
}

func (b *backends) Active() Backend {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	return b.active
}

func (b *backends) SetHealthy(backend Backend) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	knownBackend := b.unsafeSetHealth(backend, true)
	b.logger.Info("Backend became healthy again.")
	if b.active == nil {
		b.active = knownBackend
		if b.active != nil {
			b.logger.Info("Recovering from down cluster, new active backend...")
			select {
			case b.activeChan <- struct{}{}:
			default:
			}
		}
	}
}

func (b *backends) SetUnhealthy(backend Backend) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	knownBackend := b.unsafeSetHealth(backend, false)
	if b.active == knownBackend {
		b.active = b.unsafeNextHealthy()
		b.logger.Info("Active backend became unhealthy. Switching over to next available...")
		if b.active == nil {
			b.logger.Info("All backends unhealthy! No currently active backend.")
			select {
			case b.inactiveChan <- struct{}{}:
			default:
			}
		} else {
			b.logger.Info("Successfully failed over to next available backend!")
		}
	}
}

func (b *backends) unsafeNextHealthy() Backend {
	for backend, healthy := range b.all {
		if healthy {
			return backend
		}
	}
	return nil
}

func (b *backends) unsafeSetHealth(backend Backend, healthy bool) Backend {
	_, found := b.all[backend]
	if found {
		b.all[backend] = healthy
		return backend
	}
	return nil
}
