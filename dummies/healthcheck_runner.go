package dummies

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/cloudfoundry-incubator/switchboard/config"
	"github.com/onsi/ginkgo"
)

type HealthcheckRunner struct {
	sync.Mutex
	port       uint
	stopped    chan interface{}
	statusCode int
	hang       bool
}

func NewHealthcheckRunner(backend config.Backend) *HealthcheckRunner {
	return &HealthcheckRunner{
		port:       backend.HealthcheckPort,
		stopped:    make(chan interface{}),
		statusCode: http.StatusOK,
		hang:       false,
	}
}

func (fh *HealthcheckRunner) SetHang(hang bool) {
	fh.Lock()
	defer fh.Unlock()

	fh.hang = hang
}

func (fh *HealthcheckRunner) SetStatusCode(statusCode int) {
	fh.Lock()
	defer fh.Unlock()

	fh.statusCode = statusCode
}

func (fh *HealthcheckRunner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	errChan := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/", fh.health)

	server := http.Server{
		Handler: mux,
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", fh.port))
	if err != nil {
		return err
	}

	go func() {
		err := server.Serve(listener)
		if err != nil {
			errChan <- err
		}
		close(fh.stopped)
	}()

	close(ready)

	for {
		select {
		case err := <-errChan:
			fmt.Fprintf(ginkgo.GinkgoWriter, "Error stopping healthcheck: %v\n", err)
			return err
		case <-signals:
			err := listener.Close()
			if err != nil {
				errChan <- err
			} else {
				<-fh.stopped
				return nil
			}
		}
	}
}

func (fh *HealthcheckRunner) health(w http.ResponseWriter, req *http.Request) {
	fh.Lock()
	defer fh.Unlock()

	if fh.hang {
		select {}
	}

	w.WriteHeader(fh.statusCode)
	switch fh.statusCode {
	case http.StatusOK:
		io.WriteString(w, "synced")
	case http.StatusServiceUnavailable:
		io.WriteString(w, "")
	}
}
