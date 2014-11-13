package switchboard

import (
	"sync"

	"github.com/pivotal-golang/lager"
)

type Backends interface {
	All() <-chan Backend
	SetActive(backend Backend) error
	Active() Backend
	SetHealthy(backend Backend)
	SetUnhealthy(backend Backend)
	Healthy() <-chan Backend
}

type backends struct {
	mutex  sync.Mutex
	all    []*statefulBackend
	active *statefulBackend
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

	b.active = b.all[0]

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

	return b.active.backend
}

func (b *backends) SetActive(backend Backend) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	// // once healthy is implemented, use that instead of all
	// idx := unsafeIndexOf(b.all, backend)
	// if idx == -1 {
	// 	return errors.New("Unknown backend")
	// }

	b.active = &statefulBackend{
		backend: backend,
		healthy: true,
	}

	return nil
}

func (b *backends) SetHealthy(backend Backend) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.setHealth(backend, true)
}

func (b *backends) SetUnhealthy(backend Backend) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.setHealth(backend, false)
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

func (b *backends) setHealth(backend Backend, healthy bool) {
	for _, sb := range b.all {
		if sb.backend == backend {
			sb.healthy = healthy
			break
		}
	}
}

// func unsafeIndexOf(b []Backend, backend Backend) int {
//   index := -1
//   for i, aBackend := range b.all {
//     if aBackend == backend {
//       index = i
//       break
//     }
//   }
//   return index
// }
