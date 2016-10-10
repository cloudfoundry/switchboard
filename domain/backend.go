package domain

import (
	"errors"
	"fmt"
	"net"

	"sync"

	"code.cloudfoundry.org/lager"
)

var BridgesProvider = NewBridges
var Dialer = net.Dial

type Backend struct {
	mutex          sync.RWMutex
	host           string
	port           uint
	statusPort     uint
	statusEndpoint string
	logger         lager.Logger
	bridges        Bridges
	name           string
	healthy        bool
}

type BackendJSON struct {
	Host                string `json:"host"`
	Port                uint   `json:"port"`
	StatusPort          uint   `json:"status_port"`
	Healthy             bool   `json:"healthy"`
	Name                string `json:"name"`
	CurrentSessionCount uint   `json:"currentSessionCount"`
}

func NewBackend(
	name string,
	host string,
	port uint,
	statusPort uint,
	statusEndpoint string,
	logger lager.Logger) *Backend {

	return &Backend{
		name:           name,
		host:           host,
		port:           port,
		statusPort:     statusPort,
		statusEndpoint: statusEndpoint,
		logger:         logger,
		bridges:        BridgesProvider(logger),
	}
}

func (b *Backend) HealthcheckUrl() string {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return fmt.Sprintf("http://%s:%d/%s", b.host, b.statusPort, b.statusEndpoint)
}

func (b *Backend) Bridge(clientConn net.Conn) error {
	backendAddr := fmt.Sprintf("%s:%d", b.host, b.port)

	backendConn, err := Dialer("tcp", backendAddr)
	if err != nil {
		return errors.New(fmt.Sprintf("Error establishing connection to backend: %s", err))
	}

	bridge := b.bridges.Create(clientConn, backendConn)
	bridge.Connect()
	_ = b.bridges.Remove(bridge) //untested

	return nil
}

func (b *Backend) SeverConnections() {
	b.logger.Info(fmt.Sprintf("Severing all connections to %s at %s:%d", b.name, b.host, b.port))
	b.bridges.RemoveAndCloseAll()
}

func (b *Backend) SetHealthy() {
	if !b.healthy {
		b.logger.Info("Previously unhealthy backend became healthy.", lager.Data{"backend": b.AsJSON()})
	}

	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.healthy = true
}

func (b *Backend) SetUnhealthy() {
	if b.healthy {
		b.logger.Info("Previously healthy backend became unhealthy.", lager.Data{"backend": b.AsJSON()})
	}

	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.healthy = false
}

func (b *Backend) Healthy() bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return b.healthy
}

func (b *Backend) AsJSON() BackendJSON {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return BackendJSON{
		Host:                b.host,
		Port:                b.port,
		StatusPort:          b.statusPort,
		Name:                b.name,
		Healthy:             b.healthy,
		CurrentSessionCount: b.bridges.Size(),
	}
}
