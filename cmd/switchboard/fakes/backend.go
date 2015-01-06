package fakes

import (
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/onsi/ginkgo"
)

type FakeBackend struct {
	port            uint
	healthcheckPort uint
}

func NewFakeBackend(port, healthcheckPort uint) *FakeBackend {
	return &FakeBackend{
		port:            port,
		healthcheckPort: healthcheckPort,
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
	var conn net.Conn
	go func() {
		for {
			conn, err = l.Accept()
			if err != nil {
				errChan <- errors.New(fmt.Sprintf("Error accepting: ", err.Error()))
			} else {
				defer conn.Close()
				go fb.handleRequest(conn)
			}
		}
	}()

	fmt.Fprintf(ginkgo.GinkgoWriter, "Backend listening on port %s\n", address)
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
			fmt.Fprintln(ginkgo.GinkgoWriter, "Dummy backend received on connection: "+string(data))
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
			response := fmt.Sprintf(
				`{"BackendPort": %d, "HealthcheckPort": %d, "Message": "%s"}`,
				fb.port,
				fb.healthcheckPort,
				string(data),
			)
			fmt.Fprintln(ginkgo.GinkgoWriter, "Dummy backend writing to connection: Echo: "+response)
			conn.Write([]byte(response))
		case err := <-errCh:
			fmt.Fprintln(ginkgo.GinkgoWriter, "Error: "+err.Error())
			conn.Close()
			break
		}
	}
}
