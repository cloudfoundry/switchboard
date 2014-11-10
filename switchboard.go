package switchboard

import (
	"fmt"
	"log"
	"net"

	"github.com/pivotal-golang/lager"
)

type Switchboard struct {
	logger   lager.Logger
	listener net.Listener
	backends Backends
}

func New(listener net.Listener, backends Backends, logger lager.Logger) Switchboard {
	return Switchboard{
		logger:   logger,
		listener: listener,
		backends: backends,
	}
}

func (bm *Switchboard) Run() {
	bm.backends.StartHealthchecks()
	for {
		clientConn, err := bm.listener.Accept()
		if err != nil {
			log.Fatal(fmt.Sprintf("Error accepting client connection: %v", err))
		}

		backend := bm.backends.CurrentBackend()
		backendConn, err := backend.Dial()
		if err != nil {
			bm.logger.Error("Error connection to backend.", err)
			return
		}

		bridge := NewConnectionBridge(clientConn, backendConn, bm.logger)
		backend.AddBridge(bridge)

		go func() {
			bridge.Connect()
			backend.RemoveBridge(bridge)
		}()
	}
}
