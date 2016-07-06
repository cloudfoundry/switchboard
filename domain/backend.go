package domain

import (
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/pivotal-golang/lager"
)

var BridgesProvider = NewBridges
var Dialer = net.Dial

//go:generate counterfeiter . Backend
type Backend interface {
	HealthcheckUrl() string
	Bridge(clientConn net.Conn) error
	SeverConnections()
	AsJSON() BackendJSON
	EnableTraffic()
	DisableTraffic()
	TrafficEnabled() bool
}

type backend struct {
	mutex           sync.RWMutex
	host            string
	port            uint
	healthcheckPort uint
	logger          lager.Logger
	bridges         Bridges
	name            string
	trafficEnabled  bool
}

type BackendJSON struct {
	Host                string `json:"host"`
	Port                uint   `json:"port"`
	Healthy             bool   `json:"healthy"`
	Active              bool   `json:"active"`
	Name                string `json:"name"`
	CurrentSessionCount uint   `json:"currentSessionCount"`
	TrafficEnabled      bool   `json:"trafficEnabled"`
}

func NewBackend(
	name string,
	host string,
	port uint,
	healthcheckPort uint,
	logger lager.Logger,
) Backend {
	return &backend{
		name:            name,
		host:            host,
		port:            port,
		healthcheckPort: healthcheckPort,
		logger:          logger,
		bridges:         BridgesProvider(logger),
		trafficEnabled:  true,
	}
}

func (b *backend) HealthcheckUrl() string {
	return fmt.Sprintf("http://%s:%d", b.host, b.healthcheckPort)
}

func (b *backend) TrafficEnabled() bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return b.trafficEnabled
}

func (b *backend) Bridge(clientConn net.Conn) error {
	backendAddr := fmt.Sprintf("%s:%d", b.host, b.port)

	b.mutex.RLock()
	trafficEnabled := b.trafficEnabled
	b.mutex.RUnlock()

	if !trafficEnabled {
		b.logger.Info(fmt.Sprintf("Traffic disabled - not routing to %s at %s:%d", b.name, b.host, b.port))
		err := clientConn.Close()
		return err
	}

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

func (b *backend) SeverConnections() {
	b.logger.Info(fmt.Sprintf("Severing all connections to %s at %s:%d", b.name, b.host, b.port))
	b.bridges.RemoveAndCloseAll()
}

func (b *backend) AsJSON() BackendJSON {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return BackendJSON{
		Host:                b.host,
		Port:                b.port,
		Name:                b.name,
		CurrentSessionCount: b.bridges.Size(),
		TrafficEnabled:      b.trafficEnabled,
	}
}

func (b *backend) EnableTraffic() {
	b.logger.Info(fmt.Sprintf("Enabling traffic for backend %s at %s:%d", b.name, b.host, b.port))

	b.mutex.Lock()
	b.trafficEnabled = true
	b.mutex.Unlock()
}

func (b *backend) DisableTraffic() {
	b.logger.Info(fmt.Sprintf("Disabling traffic for backend %s at %s:%d", b.name, b.host, b.port))

	b.mutex.Lock()
	b.trafficEnabled = false
	b.mutex.Unlock()

	b.SeverConnections()
}
