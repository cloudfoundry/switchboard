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

//go:generate counterfeiter -o domainfakes/fake_net_conn.go /usr/local/opt/go/libexec/src/net/net.go Conn
//go:generate counterfeiter . Backend
type Backend interface {
	HealthcheckUrl() string
	Bridge(clientConn net.Conn) error
	SeverConnections()
	AsJSON() BackendJSON
}

type backend struct {
	mutex          sync.RWMutex
	host           string
	port           uint
	statusPort     uint
	statusEndpoint string
	logger         lager.Logger
	bridges        Bridges
	name           string
}

type BackendJSON struct {
	Host                string `json:"host"`
	Port                uint   `json:"port"`
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
	logger lager.Logger) Backend {

	return &backend{
		name:           name,
		host:           host,
		port:           port,
		statusPort:     statusPort,
		statusEndpoint: statusEndpoint,
		logger:         logger,
		bridges:        BridgesProvider(logger),
	}
}

func (b *backend) HealthcheckUrl() string {
	return fmt.Sprintf("http://%s:%d/%s", b.host, b.statusPort, b.statusEndpoint)
}

func (b *backend) Bridge(clientConn net.Conn) error {
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

func (b *backend) SeverConnections() {
	b.logger.Info(fmt.Sprintf("Severing all connections to %s at %s:%d", b.name, b.host, b.port))
	b.bridges.RemoveAndCloseAll()
}

func (b *backend) AsJSON() BackendJSON {
	return BackendJSON{
		Host:                b.host,
		Port:                b.port,
		Name:                b.name,
		CurrentSessionCount: b.bridges.Size(),
	}
}
