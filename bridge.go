package switchboard

import (
	"io"

	"github.com/pivotal-golang/lager"
)

type Bridge interface {
	Connect()
	Close()
}

type ConnectionBridge struct {
	done    chan struct{}
	Client  io.ReadWriteCloser
	Backend io.ReadWriteCloser
	logger  lager.Logger
}

func NewConnectionBridge(client, backend io.ReadWriteCloser, logger lager.Logger) Bridge {
	return &ConnectionBridge{
		done:    make(chan struct{}),
		Client:  client,
		Backend: backend,
		logger:  logger,
	}
}

func (b ConnectionBridge) Connect() {
	defer b.Client.Close()
	defer b.Backend.Close()

	select {
	case <-b.safeCopy(b.Client, b.Backend):
	case <-b.safeCopy(b.Backend, b.Client):
	case <-b.done:
	}
}

func (b ConnectionBridge) Close() {
	close(b.done)
}

func (b ConnectionBridge) safeCopy(from, to io.ReadWriteCloser) chan struct{} {
	copyDone := make(chan struct{})
	go func() {
		_, err := io.Copy(from, to)
		if err != nil {
			b.logger.Error("Error copying from 'from' to 'to'", err)
		} else {
			b.logger.Info("Copying from 'from' to 'to' completed without an error\n")
		}
		close(copyDone)
	}()
	return copyDone
}
