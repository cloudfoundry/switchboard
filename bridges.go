package switchboard

import "errors"

type Bridges interface {
	RemoveBridge(bridge Bridge) error
	RemoveAndCloseAllBridges()
	AddBridge(bridge Bridge)
	Size() int
	IndexOfBridge(bridge Bridge) (int, error)
}

type bridges struct {
	bridges []Bridge
}

func NewBridges() Bridges {
	return &bridges{}
}

func (b *bridges) RemoveBridge(bridge Bridge) error {
	index, err := b.IndexOfBridge(bridge)
	if err != nil {
		return err
	}
	b.removeBridgeAt(index)
	return nil
}

func (b *bridges) RemoveAndCloseAllBridges() {
	for _, bridge := range b.bridges {
		bridge.Close()
	}
	b.bridges = []Bridge{}
}

func (b *bridges) AddBridge(bridge Bridge) {
	b.bridges = append(b.bridges, bridge)
}

func (b *bridges) Size() int {
	return len(b.bridges)
}

func (b *bridges) IndexOfBridge(bridge Bridge) (int, error) {
	index := -1
	for i, aBridge := range b.bridges {
		if aBridge == bridge {
			index = i
			break
		}
	}
	if index == -1 {
		return -1, errors.New("Bridge not found")
	}
	return index, nil
}

func (b *bridges) removeBridgeAt(index int) {
	copy(b.bridges[index:], b.bridges[index+1:])
	b.bridges = b.bridges[:len(b.bridges)-1]
}
