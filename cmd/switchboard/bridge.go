package main

import (
	"fmt"
	"io"
	"net"
)

type Bridge struct {
	done    chan struct{}
	Client  net.Conn
	Backend net.Conn
}

func NewBridge(client, backend net.Conn) Bridge {
	return Bridge{
		done:    make(chan struct{}),
		Client:  client,
		Backend: backend,
	}
}

func (b Bridge) Connect() {
	defer b.Client.Close()
	defer b.Backend.Close()

	select {
	case <-safeCopy(b.Client, b.Backend):
	case <-safeCopy(b.Backend, b.Client):
	case <-b.done:
	}
}

func safeCopy(from, to net.Conn) chan struct{} {
	done := make(chan struct{})
	go func() {
		_, err := io.Copy(from, to)
		if err != nil {
			fmt.Printf("Error copying from 'from' to 'to': %v\n", err.Error())
		} else {
			fmt.Printf("Copying from 'from' to 'to' completed without an error\n")
		}
		close(done)
	}()
	return done
}

func (b Bridge) Close() {
	close(b.done)
}
