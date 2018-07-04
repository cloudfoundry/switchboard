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
	endpoint   string
	stopped    chan interface{}
	statusCode int
	index      int
	hang       bool
}

func NewHealthcheckRunner(backend config.Backend, index int) *HealthcheckRunner {
	return &HealthcheckRunner{
		port:       backend.StatusPort,
		endpoint:   backend.StatusEndpoint,
		stopped:    make(chan interface{}),
		statusCode: http.StatusOK,
		index:      index,
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
	mux.HandleFunc(fmt.Sprintf("/%s", fh.endpoint), fh.health)

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
		io.WriteString(w, fmt.Sprintf(`{"wsrep_local_state":4,"wsrep_local_state_comment":"Synced","wsrep_local_index":%d,"healthy":true}`, fh.index))
	case http.StatusServiceUnavailable:
		io.WriteString(w, "")
	}
}
