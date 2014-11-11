package switchboard

import (
	"errors"
	"sync"
)

type Bridges interface {
	Add(bridge Bridge)
	Remove(bridge Bridge) error
	RemoveAndCloseAll()
	Size() int
	Contains(bridge Bridge) bool
}

type concurrentBridges struct {
	mutex   sync.Mutex
	bridges []Bridge
}

func NewBridges() Bridges {
	return &concurrentBridges{}
}

func (b *concurrentBridges) Add(bridge Bridge) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.bridges = append(b.bridges, bridge)
}

func (b *concurrentBridges) Remove(bridge Bridge) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if !b.unsafeContains(bridge) {
		return errors.New("Bridge not found")
	}

	index := b.unsafeIndexOf(bridge)
	copy(b.bridges[index:], b.bridges[index+1:])
	b.bridges = b.bridges[:len(b.bridges)-1]

	return nil
}

func (b *concurrentBridges) RemoveAndCloseAll() {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	for _, bridge := range b.bridges {
		bridge.Close()
	}
	b.bridges = []Bridge{}
}

func (b *concurrentBridges) Size() int {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	return len(b.bridges)
}

func (b *concurrentBridges) Contains(bridge Bridge) bool {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	return b.unsafeContains(bridge)
}

func (b *concurrentBridges) unsafeContains(bridge Bridge) bool {
	return b.unsafeIndexOf(bridge) != -1
}

func (b *concurrentBridges) unsafeIndexOf(bridge Bridge) int {
	index := -1
	for i, aBridge := range b.bridges {
		if aBridge == bridge {
			index = i
			break
		}
	}
	return index
}
