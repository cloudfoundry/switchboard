package switchboard

import (
	"sync"

	"github.com/pivotal-golang/lager"
)

type Backends interface {
	All() <-chan Backend
	Active() Backend
	SetHealthy(backend Backend)
	SetUnhealthy(backend Backend)
	Healthy() <-chan Backend
}

type backends struct {
	mutex  sync.Mutex
	all    []*statefulBackend
	active Backend
	logger lager.Logger
}

type statefulBackend struct {
	backend Backend
	healthy bool
}

func NewBackends(backendIPs []string, backendPorts []uint, healthcheckPorts []uint, logger lager.Logger) Backends {
	b := &backends{
		logger: logger,
		all:    make([]*statefulBackend, len(backendIPs)),
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
	}

	return b
}

func (b *backends) All() <-chan Backend {
	ch := make(chan Backend, len(b.all))

	go func() {
		b.mutex.Lock()
		defer b.mutex.Unlock()

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
	knownBackend := b.setHealth(backend, true)
	if b.active == nil {
		b.active = knownBackend
	}
}

func (b *backends) SetUnhealthy(backend Backend) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	knownBackend := b.setHealth(backend, false)
	if b.active == knownBackend {
		b.active = b.nextHealthy()
	}
}

func (b *backends) nextHealthy() Backend {
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
		b.mutex.Lock()
		defer b.mutex.Unlock()

		for _, sb := range b.all {
			if sb.healthy {
				c <- sb.backend
			}
		}

		close(c)
	}()

	return c
}

func (b *backends) setHealth(backend Backend, healthy bool) Backend {
	for _, sb := range b.all {
		if sb.backend == backend {
			sb.healthy = healthy
			return sb.backend
		}
	}
	return nil
}
