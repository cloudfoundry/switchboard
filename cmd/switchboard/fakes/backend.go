package fakes

import (
	"errors"
	"fmt"
	"net"
	"os"
)

type FakeBackend struct {
	port uint
}

func NewFakeBackend(port uint) *FakeBackend {
	return &FakeBackend{
		port: port,
	}
}

func (fb *FakeBackend) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	address := fmt.Sprintf("%s:%d", "localhost", fb.port)

	l, err := net.Listen("tcp", address)
	if err != nil {
		return errors.New(fmt.Sprintf("Backend error listening on address: %s - %s\n", address, err.Error()))
	}
	defer l.Close()

	errChan := make(chan error, 1)
	go func() {
		for {
			conn, err := l.Accept()
			defer conn.Close()
			if err != nil {
				errChan <- errors.New(fmt.Sprintf("Error accepting: ", err.Error()))
			}
			go fb.handleRequest(conn)
		}
	}()

	fmt.Printf("Backend listening on port %s\n", address)
	close(ready)

	for {
		select {
		case err := <-errChan:
			return err
		case <-signals:
			l.Close()
		}
	}
	return nil
}

func (fb *FakeBackend) handleRequest(conn net.Conn) {
	dataCh := make(chan []byte)
	errCh := make(chan error)

	go func(ch chan []byte, eCh chan error) {
		for {
			data := make([]byte, 1024)
			n, err := conn.Read(data)
			fmt.Println("Dummy backend received on connection: " + string(data))
			if err != nil {
				eCh <- err
				return
			}
			ch <- data[:n]
		}
	}(dataCh, errCh)

	for {
		select {
		case data := <-dataCh:
			fmt.Println("Dummy backend writing to connection: Echo: " + string(data))
			conn.Write([]byte(fmt.Sprintf("Echo from port %d: %s", fb.port, string(data))))
		case err := <-errCh:
			fmt.Println("Error: " + err.Error())
			conn.Close()
			break
		}
	}
}
