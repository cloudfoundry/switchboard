package domain

import (
	"fmt"
	"io"
	"net"

	"github.com/pivotal-golang/lager"
)

type Bridge struct {
	done            chan struct{}
	Client, Backend net.Conn
	Logger          lager.Logger
}

//go:generate counterfeiter -o domainfakes/fake_net_conn.go /usr/local/opt/go/libexec/src/net/net.go Conn
func NewBridge(client, backend net.Conn, logger lager.Logger) *Bridge {
	return &Bridge{
		done:    make(chan struct{}),
		Client:  client,
		Backend: backend,
		Logger:  logger,
	}
}

func (b Bridge) Connect() {
	b.Logger.Info(fmt.Sprintf("Session established %s", b))

	defer b.Logger.Info(fmt.Sprintf("Session closed %s", b)) // defers are LIFO
	defer b.Client.Close()
	defer b.Backend.Close()

	select {
	case <-b.safeCopy(b.Client, b.Backend):
	case <-b.safeCopy(b.Backend, b.Client):
	case <-b.done:
	}
}

func (b Bridge) Close() {
	close(b.done)
}

func (b Bridge) safeCopy(from, to net.Conn) chan struct{} {
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

func (b Bridge) String() string {
	return fmt.Sprintf("from client at %v to backend at %v", b.Client.RemoteAddr(), b.Backend.RemoteAddr())
}
