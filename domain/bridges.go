package domain

import (
	"errors"
	"net"
	"sync"

	"github.com/pivotal-golang/lager"
)

var BridgeProvider = NewBridge

type Bridges interface {
	Create(clientConn, backendConn net.Conn) Bridge
	Remove(bridge Bridge) error
	RemoveAndCloseAll()
	Size() uint
	Contains(bridge Bridge) bool
}

type concurrentBridges struct {
	mutex   sync.RWMutex
	bridges []Bridge
	logger  lager.Logger
}

func NewBridges(logger lager.Logger) Bridges {
	return &concurrentBridges{
		logger: logger,
	}
}

func (b *concurrentBridges) Create(clientConn, backendConn net.Conn) Bridge {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	bridge := BridgeProvider(clientConn, backendConn, b.logger)
	b.bridges = append(b.bridges, bridge)
	return bridge
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

func (b *concurrentBridges) Size() uint {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return uint(len(b.bridges))
}

func (b *concurrentBridges) Contains(bridge Bridge) bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

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
