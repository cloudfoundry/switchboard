package domain

import (
	"errors"
	"net"
	"sync"

	"github.com/pivotal-golang/lager"
	"github.com/cloudfoundry-incubator/switchboard/models"
)

var BridgeProvider = func(client, backend net.Conn, logger lager.Logger) models.Bridge {
	return NewBridge(client, backend, logger)
}

type ConcurrentBridges struct {
	mutex   sync.RWMutex
	bridges []models.Bridge
	Logger  lager.Logger
}

func NewBridges(logger lager.Logger) *ConcurrentBridges {
	return &ConcurrentBridges{
		Logger: logger,
	}
}

func (b *ConcurrentBridges) Create(clientConn, backendConn net.Conn) models.Bridge {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	bridge := BridgeProvider(clientConn, backendConn, b.Logger)
	b.bridges = append(b.bridges, bridge)
	return bridge
}

func (b *ConcurrentBridges) Remove(bridge models.Bridge) error {
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

func (b *ConcurrentBridges) RemoveAndCloseAll() {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	for _, bridge := range b.bridges {
		bridge.Close()
	}
	b.bridges = nil
}

func (b *ConcurrentBridges) Size() uint {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return uint(len(b.bridges))
}

func (b *ConcurrentBridges) Contains(bridge models.Bridge) bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return b.unsafeContains(bridge)
}

func (b *ConcurrentBridges) unsafeContains(bridge models.Bridge) bool {
	return b.unsafeIndexOf(bridge) != -1
}

func (b *ConcurrentBridges) unsafeIndexOf(bridge models.Bridge) int {
	index := -1
	for i, aBridge := range b.bridges {
		if aBridge == bridge {
			index = i
			break
		}
	}
	return index
}
