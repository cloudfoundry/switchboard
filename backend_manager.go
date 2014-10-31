package switchboard

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/pivotal-golang/lager"
)

type BackendInfo struct {
	Port            uint
	HealthcheckPort uint
	Ip              string
}

type BackendManager struct {
	Logger             lager.Logger
	HealthcheckTimeout time.Duration
	BackendInfo        BackendInfo
	Listener           net.Listener
}

func acceptClientConnection(l net.Listener) net.Conn {
	clientConn, err := l.Accept()
	if err != nil {
		log.Fatal("Error accepting client connection: %v", err)
	}
	return clientConn
}

func (bm *BackendManager) Run() {
	fmt.Printf("backend info: %v", bm.BackendInfo)

	backend := NewBackend("backend1", bm.BackendInfo.Ip, bm.BackendInfo.Port)

	healthcheck := NewHttpHealthCheck(
		bm.BackendInfo.Ip,
		bm.BackendInfo.HealthcheckPort,
		bm.HealthcheckTimeout,
		bm.Logger,
	)
	healthcheck.Start(backend.RemoveAndCloseAllBridges)

	// for {
	bm.ProxyToBackend(&backend)
	// }
}

func (bm *BackendManager) ProxyToBackend(backend *Backend) {
	for {
		clientConn := acceptClientConnection(bm.Listener)
		defer clientConn.Close()

		backendConn, err := backend.Dial()
		if err != nil {
			bm.Logger.Fatal("Error connection to backend.", err)
		}
		defer backendConn.Close()

		bridge := NewConnectionBridge(clientConn, backendConn, bm.Logger)
		backend.AddBridge(bridge)

		go func() {
			bridge.Connect()
			backend.RemoveBridge(bridge)
		}()
	}
}
