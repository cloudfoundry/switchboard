package switchboard

import (
	"github.com/pivotal-golang/lager"
	"sync"
)

type Backends interface {
	All() <-chan Backend
	Active() Backend
	SetHealthy(backend Backend)
	SetUnhealthy(backend Backend)
	Healthy() <-chan Backend
	ActivityChannels() (<-chan struct{}, <-chan struct{})
}

type backends struct {
	mutex        sync.RWMutex
	all          []*statefulBackend
	active       Backend
	logger       lager.Logger
	activeChan   chan struct{}
	inactiveChan chan struct{}
}

type statefulBackend struct {
	backend Backend
	healthy bool
}

func NewBackends(backendIPs []string, backendPorts []uint, healthcheckPorts []uint, logger lager.Logger) Backends {
	b := &backends{
		logger:       logger,
		all:          make([]*statefulBackend, len(backendIPs)),
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

		b.all[i] = &statefulBackend{
			backend: backend,
			healthy: true,
		}
	}

	if len(b.all) > 0 {
		b.active = b.all[0].backend
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

func (b *backends) All() <-chan Backend {
	ch := make(chan Backend, len(b.all))

	go func() {
		b.mutex.RLock()
		defer b.mutex.RUnlock()

		for _, sb := range b.all {
			ch <- sb.backend
		}
		close(ch)
	}()

	return ch
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
	for _, sb := range b.all {
		if sb.healthy {
			return sb.backend
		}
	}
	return nil
}

func (b *backends) Healthy() <-chan Backend {
	c := make(chan Backend, len(b.all))

	go func() {
		b.mutex.RLock()
		defer b.mutex.RUnlock()

		for _, sb := range b.all {
			if sb.healthy {
				c <- sb.backend
			}
		}

		close(c)
	}()

	return c
}

func (b *backends) unsafeSetHealth(backend Backend, healthy bool) Backend {
	for _, sb := range b.all {
		if sb.backend == backend {
			sb.healthy = healthy
			return sb.backend
		}
	}
	return nil
}
