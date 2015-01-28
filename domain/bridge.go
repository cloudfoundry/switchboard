package domain

import (
	"fmt"
	"io"
	"net"

	"github.com/pivotal-golang/lager"
)

type Bridge interface {
	Connect()
	Close()
}

type bridge struct {
	done            chan struct{}
	client, backend net.Conn
	logger          lager.Logger
}

func NewBridge(client, backend net.Conn, logger lager.Logger) Bridge {
	return &bridge{
		done:    make(chan struct{}),
		client:  client,
		backend: backend,
		logger:  logger,
	}
}

func (b bridge) Connect() {
	defer b.client.Close()
	defer b.backend.Close()

	select {
	case <-b.safeCopy(b.client, b.backend):
	case <-b.safeCopy(b.backend, b.client):
	case <-b.done:
	}
	b.logger.Info(fmt.Sprintf("Session closed for client at %s to backend at %s", b.client.RemoteAddr().String(), b.backend.RemoteAddr().String()))
}

func (b bridge) Close() {
	close(b.done)
}

func (b bridge) safeCopy(from, to net.Conn) chan struct{} {
	copyDone := make(chan struct{})
	go func() {
		io.Copy(from, to)
		close(copyDone)
	}()
	return copyDone
}
