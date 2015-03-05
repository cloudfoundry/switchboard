package domain

import (
	"errors"
	"fmt"
	"net"

	"github.com/pivotal-golang/lager"
)

var BridgesProvider = NewBridges
var Dialer = net.Dial

type Backend interface {
	HealthcheckUrl() string
	Bridge(clientConn net.Conn) error
	SeverConnections()
	AsJSON() BackendJSON
}

type backend struct {
	host            string
	port            uint
	healthcheckPort uint
	logger          lager.Logger
	bridges         Bridges
	name            string
}

type BackendJSON struct {
	Host                string `json:"host"`
	Port                uint   `json:"port"`
	Healthy             bool   `json:"healthy"`
	Active              bool   `json:"active"`
	Name                string `json:"name"`
	CurrentSessionCount uint   `json:"currentSessionCount"`
}

func NewBackend(
	name string,
	host string,
	port uint,
	healthcheckPort uint,
	logger lager.Logger) Backend {

	return &backend{
		name:            name,
		host:            host,
		port:            port,
		healthcheckPort: healthcheckPort,
		logger:          logger,
		bridges:         BridgesProvider(logger),
	}
}

func (b backend) HealthcheckUrl() string {
	return fmt.Sprintf("http://%s:%d", b.host, b.healthcheckPort)
}

func (b backend) Bridge(clientConn net.Conn) error {
	backendAddr := fmt.Sprintf("%s:%d", b.host, b.port)
	backendConn, err := Dialer("tcp", backendAddr)
	if err != nil {
		return errors.New(fmt.Sprintf("Error establishing connection to backend: %s", err))
	}

	go func() {
		bridge := b.bridges.Create(clientConn, backendConn)
		bridge.Connect()
		b.bridges.Remove(bridge)
	}()

	return nil
}

func (b backend) SeverConnections() {
	b.logger.Info(fmt.Sprintf("Severing all connections to %s at %s:%d", b.name, b.host, b.port))
	b.bridges.RemoveAndCloseAll()
}

func (b backend) AsJSON() BackendJSON {
	return BackendJSON{
		Host:                b.host,
		Port:                b.port,
		Name:                b.name,
		CurrentSessionCount: b.bridges.Size(),
	}
}
