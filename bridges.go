package switchboard

import "errors"

type Bridges interface {
	Add(bridge Bridge)
	Remove(bridge Bridge) error
	RemoveAndCloseAll()
	Size() int
	Contains(bridge Bridge) bool
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
	if !b.Contains(bridge) {
		return errors.New("Bridge not found")
	}

	index := b.indexOf(bridge)
	copy(b.bridges[index:], b.bridges[index+1:])
	b.bridges = b.bridges[:len(b.bridges)-1]

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

func (b *bridges) Contains(bridge Bridge) bool {
	return b.indexOf(bridge) != -1
}

func (b *bridges) indexOf(bridge Bridge) int {
	index := -1
	for i, aBridge := range b.bridges {
		if aBridge == bridge {
			index = i
			break
		}
	}
	return index
}
