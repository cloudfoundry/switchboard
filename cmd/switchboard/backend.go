package main

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
}

func NewBackend(desc, ipAddress string, port uint) Backend {
	return Backend{
		Desc:      desc,
		bridges:   []Bridge{},
		ipAddress: ipAddress,
		port:      port,
	}
}

func (b Backend) RemoveBridge(bridge Bridge) error {
	index, err := b.indexOfBridge(bridge)
	if err != nil {
		return err
	}
	b.removeBridgeAt(index)
	return nil
}

func (b Backend) indexOfBridge(bridge Bridge) (int, error) {
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
	b.bridges[len(b.bridges)-1] = Bridge{} // or the zero value of T
	b.bridges = b.bridges[:len(b.bridges)-1]
}

func (b *Backend) RemoveAllBridges() {
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
