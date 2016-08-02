package models

import "net"

type JSONSerializable interface {
	AsJSON() interface{}
}

//go:generate counterfeiter . Backends
type Backends interface {
	All() <-chan Backend
	Any() Backend
	Active() Backend
	SetHealthy(backend Backend)
	SetUnhealthy(backend Backend)
	Healthy() <-chan Backend
	JSONSerializable
}

//go:generate counterfeiter . Backend
type Backend interface {
	HealthcheckUrl() string
	Bridge(clientConn net.Conn) error
	SeverConnections()
	JSONSerializable
}

//go:generate counterfeiter . Bridges
type Bridges interface {
	Create(clientConn, backendConn net.Conn) Bridge
	Remove(bridge Bridge) error
	RemoveAndCloseAll()
	Size() uint
	Contains(bridge Bridge) bool
}

//go:generate counterfeiter . Bridge
type Bridge interface {
	Connect()
	Close()
}

//go:generate counterfeiter . ArpManager
type ArpManager interface {
	ClearCache(ip string) error
	IsCached(ip string) bool
}
