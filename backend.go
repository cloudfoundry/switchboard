package switchboard

import (
	"errors"
	"fmt"
	"net"
)

type Backend struct {
	bridges   []Bridge
	Desc      string
	ipAddress string
	port      uint
	hc        Healthcheck
}

func NewBackend(desc, ipAddress string, port uint, hc Healthcheck) *Backend {
	return &Backend{
		Desc:      desc,
		bridges:   []Bridge{},
		ipAddress: ipAddress,
		port:      port,
		hc:        hc,
	}
}

func (b *Backend) RemoveBridge(bridge Bridge) error {
	index, err := b.IndexOfBridge(bridge)
	if err != nil {
		return err
	}
	b.removeBridgeAt(index)
	return nil
}

func (b *Backend) StartHealthcheck() {
	b.hc.Start(b.RemoveAndCloseAllBridges)
}

func (b *Backend) RemoveAndCloseAllBridges() {
	for _, bridge := range b.bridges {
		bridge.Close()
	}
	b.bridges = []Bridge{}
}

func (b *Backend) AddBridge(bridge Bridge) {
	b.bridges = append(b.bridges, bridge)
}

func (b *Backend) Dial() (net.Conn, error) {
	addr := fmt.Sprintf("%s:%d", b.ipAddress, b.port)
	backendConn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return backendConn, nil
}

func (b *Backend) Bridges() []Bridge {
	return b.bridges
}

func (b *Backend) IndexOfBridge(bridge Bridge) (int, error) {
	index := -1
	for i, aBridge := range b.bridges {
		if aBridge == bridge {
			index = i
			break
		}
	}
	if index == -1 {
		return -1, errors.New("Bridge not found in backend")
	}
	return index, nil
}

func (b *Backend) removeBridgeAt(index int) {
	copy(b.bridges[index:], b.bridges[index+1:])
	b.bridges = b.bridges[:len(b.bridges)-1]
}
