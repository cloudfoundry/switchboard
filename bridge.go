package switchboard

import (
	"io"

	"github.com/cloudfoundry-incubator/cf-lager"
	"github.com/pivotal-golang/lager"
)

type Bridge interface {
	Connect()
	Close()
}

type bridge struct {
	done    chan struct{}
	client  io.ReadWriteCloser
	backend io.ReadWriteCloser
	logger  lager.Logger
}

func NewBridge(client, backend io.ReadWriteCloser) Bridge {
	return &bridge{
		done:    make(chan struct{}),
		client:  client,
		backend: backend,
		logger:  cf_lager.New("bridge"),
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
}

func (b bridge) Close() {
	close(b.done)
}

func (b bridge) safeCopy(from, to io.ReadWriteCloser) chan struct{} {
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
