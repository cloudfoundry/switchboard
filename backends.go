package switchboard

import (
	"sync"

	"github.com/pivotal-golang/lager"
)

type Backends interface {
	All() <-chan Backend
	SetActive(backend Backend) error
	Active() Backend
}

type backends struct {
	mutex  sync.Mutex
	all    []StatefulBackend
	active StatefulBackend
	logger lager.Logger
}

type StatefulBackend struct {
	backend Backend
	healthy bool
}

func NewBackends(backendIPs []string, backendPorts []uint, healthcheckPorts []uint, logger lager.Logger) Backends {
	b := &backends{
		logger: logger,
		all:    make([]StatefulBackend, len(backendIPs)),
	}

	for i, ip := range backendIPs {
		backend := NewBackend(
			ip,
			backendPorts[i],
			healthcheckPorts[i],
			logger,
		)

		b.all[i] = StatefulBackend{
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

	b.active = StatefulBackend{
		backend: backend,
		healthy: true,
	}

	return nil
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
