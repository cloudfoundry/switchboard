package switchboard

import (
	"fmt"
	"io"
)

type Bridge interface {
	Connect()
	Close()
}

type ConnectionBridge struct {
	done    chan struct{}
	Client  io.ReadWriteCloser
	Backend io.ReadWriteCloser
}

func NewConnectionBridge(client, backend io.ReadWriteCloser) *ConnectionBridge {
	return &ConnectionBridge{
		done:    make(chan struct{}),
		Client:  client,
		Backend: backend,
	}
}

func (b *ConnectionBridge) Connect() {
	defer b.Client.Close()
	defer b.Backend.Close()

	select {
	case <-safeCopy(b.Client, b.Backend):
	case <-safeCopy(b.Backend, b.Client):
	case <-b.done:
	}
}

func safeCopy(from, to io.ReadWriteCloser) chan struct{} {
	copyDone := make(chan struct{})
	go func() {
		_, err := io.Copy(from, to)
		if err != nil {
			fmt.Printf("Error copying from 'from' to 'to': %v\n", err.Error())
		} else {
			fmt.Printf("Copying from 'from' to 'to' completed without an error\n")
		}
		close(copyDone)
	}()
	return copyDone
}

func (b *ConnectionBridge) Close() {
	close(b.done)
}
