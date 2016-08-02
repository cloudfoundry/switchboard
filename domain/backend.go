package domain

import (
	"errors"
	"fmt"
	"net"

	"github.com/pivotal-golang/lager"
	"github.com/cloudfoundry-incubator/switchboard/models"
)

var BridgesProvider = func(logger lager.Logger) models.Bridges {
	return NewBridges(logger)
}
var Dialer = net.Dial

type Backend struct {
	Host           string
	Port           uint
	StatusPort     uint
	StatusEndpoint string
	Logger         lager.Logger
	Bridges        models.Bridges
	Name           string
}

func NewBackend(
	name string,
	host string,
	port uint,
	statusPort uint,
	statusEndpoint string,
	logger lager.Logger) *Backend {

	return &Backend{
		Name:           name,
		Host:           host,
		Port:           port,
		StatusPort:     statusPort,
		StatusEndpoint: statusEndpoint,
		Logger:         logger,
		Bridges:        BridgesProvider(logger),
	}
}

func (b *Backend) HealthcheckUrl() string {
	return fmt.Sprintf("http://%s:%d/%s", b.Host, b.StatusPort, b.StatusEndpoint)
}

func (b *Backend) Bridge(clientConn net.Conn) error {
	backendAddr := fmt.Sprintf("%s:%d", b.Host, b.Port)
	backendConn, err := Dialer("tcp", backendAddr)
	if err != nil {
		return errors.New(fmt.Sprintf("Error establishing connection to backend: %s", err))
	}

	go func() {
		bridge := b.Bridges.Create(clientConn, backendConn)
		bridge.Connect()
		b.Bridges.Remove(bridge)
	}()

	return nil
}

func (b *Backend) SeverConnections() {
	b.Logger.Info(fmt.Sprintf("Severing all connections to %s at %s:%d", b.Name, b.Host, b.Port))
	b.Bridges.RemoveAndCloseAll()
}

func (b *Backend) AsJSON() interface{} {
	return BackendJSON{
		Host:                b.Host,
		Port:                b.Port,
		Name:                b.Name,
		CurrentSessionCount: b.Bridges.Size(),
	}
}

type BackendJSON struct {
	Host                string `json:"host"`
	Port                uint   `json:"port"`
	Healthy             bool   `json:"healthy"`
	Active              bool   `json:"active"`
	Name                string `json:"name"`
	CurrentSessionCount uint   `json:"currentSessionCount"`
}
