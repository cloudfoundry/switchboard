package switchboard

import "errors"

type Bridges interface {
	Add(bridge Bridge)
	Remove(bridge Bridge) error
	RemoveAndCloseAll()
	Size() int
	Index(bridge Bridge) (int, error)
}

type bridges struct {
	bridges []Bridge
}

func NewBridges() Bridges {
	return &bridges{}
}

func (b *bridges) Add(bridge Bridge) {
	b.bridges = append(b.bridges, bridge)
}

func (b *bridges) Remove(bridge Bridge) error {
	index, err := b.Index(bridge)
	if err != nil {
		return err
	}
	b.removeBridgeAt(index)
	return nil
}

func (b *bridges) RemoveAndCloseAll() {
	for _, bridge := range b.bridges {
		bridge.Close()
	}
	b.bridges = []Bridge{}
}

func (b *bridges) Size() int {
	return len(b.bridges)
}

func (b *bridges) Index(bridge Bridge) (int, error) {
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
