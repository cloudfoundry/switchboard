package switchboard

import (
	"errors"
	"fmt"
	"net"

	"github.com/pivotal-golang/lager"
)

type Backend interface {
	HealthcheckUrl() string
	Bridge(clientConn net.Conn) error
	Dial() (net.Conn, error)
	SeverConnections()
}

type backend struct {
	ipAddress       string
	port            uint
	healthcheckPort uint
	logger          lager.Logger
	bridges         Bridges
}

func NewBackend(ipAddress string, port uint, healthcheckPort uint, logger lager.Logger) Backend {
	return &backend{
		ipAddress:       ipAddress,
		port:            port,
		healthcheckPort: healthcheckPort,
		logger:          logger,
		bridges:         NewBridges(),
	}
}

func (b backend) HealthcheckUrl() string {
	endpoint := fmt.Sprintf("http://%s:%d", b.ipAddress, b.healthcheckPort)
	return endpoint
}

func (b backend) Bridge(clientConn net.Conn) error {
	backendConn, err := b.Dial()
	if err != nil {
		return errors.New(fmt.Sprintf("Error connection to backend: %v", err))
	}

	bridge := NewConnectionBridge(clientConn, backendConn, b.logger)
	b.bridges.Add(bridge)

	go func() {
		bridge.Connect()
		b.bridges.Remove(bridge)
	}()

	return nil
}

func (b backend) Dial() (net.Conn, error) {
	addr := fmt.Sprintf("%s:%d", b.ipAddress, b.port)
	backendConn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return backendConn, nil
}

func (b backend) SeverConnections() {
	b.bridges.RemoveAndCloseAll()
}
