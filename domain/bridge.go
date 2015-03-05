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
	b.logger.Info(fmt.Sprintf("Session established %s", b))

	defer b.logger.Info(fmt.Sprintf("Session closed %s", b)) // defers are LIFO
	defer b.client.Close()
	defer b.backend.Close()

	select {
	case <-b.safeCopy(b.client, b.backend):
	case <-b.safeCopy(b.backend, b.client):
	case <-b.done:
	}
}

func (b bridge) Close() {
	close(b.done)
}

func (b bridge) safeCopy(from, to net.Conn) chan struct{} {
	copyDone := make(chan struct{})
	go func() {
		// We don't want to capture the error because it's not meaningful -
		// whenever a connection is closed, one half will return without error
		// but the other half will return an error.

		// A more elegant solution might involve sending the error down a channel
		// and correlating it to the (expected) closure of the other half of the
		// channel. If it can't correlate then we have an actual error,
		// otherwise we can safely ignore it.
		_, _ = io.Copy(from, to)

		close(copyDone)
	}()
	return copyDone
}

func (b bridge) String() string {
	return fmt.Sprintf("from client at %v to backend at %v", b.client.RemoteAddr(), b.backend.RemoteAddr())
}
