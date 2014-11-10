package switchboard

import (
	"fmt"
	"log"
	"net"

	"github.com/pivotal-golang/lager"
)

type Switchboard struct {
	Logger   lager.Logger
	Listener net.Listener
	Backends []*Backend
}

func acceptClientConnection(l net.Listener) net.Conn {
	clientConn, err := l.Accept()
	if err != nil {
		log.Fatal(fmt.Sprintf("Error accepting client connection: %v", err))
	}
	return clientConn
}

func (bm *Switchboard) Run() {
	for _, backend := range bm.Backends {
		backend.StartHealthcheck()
	}
	bm.proxyToBackend()
}

func (bm *Switchboard) proxyToBackend() {
	for {
		clientConn := acceptClientConnection(bm.Listener)
		defer clientConn.Close()

		backend := bm.getCurrentBackend()
		backendConn, err := backend.Dial()
		if err != nil {
			bm.Logger.Error("Error connection to backend.", err)
			return
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

func (bm *Switchboard) getCurrentBackend() *Backend {
	currentBackendIndex := 0
	backend := bm.Backends[currentBackendIndex]
	// bm.Logger.Info(fmt.Sprintf("Failing over from %s to next available backend", backend.Desc))
	// currentBackendIndex++
	// currentBackendIndex = currentBackendIndex % len(backends)
	return backend
}
