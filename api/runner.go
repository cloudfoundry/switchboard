package api

import (
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/pivotal-cf-experimental/switchboard/domain"
	"github.com/pivotal-golang/lager"
)

type Runner struct {
	logger   lager.Logger
	port     uint
	backends domain.Backends
}

func NewRunner(port uint, backends domain.Backends, logger lager.Logger) Runner {
	return Runner{
		logger:   logger,
		port:     port,
		backends: backends,
	}
}

func (a Runner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	http.HandleFunc("/v0/backends", backendsIndex(a.backends))

	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", a.port))
	if err != nil {
		return err
	} else {
		a.logger.Info(fmt.Sprintf("Proxy api listening on port %d\n", a.port))
	}

	errChan := make(chan error)
	go func() {
		err := http.Serve(listener, nil)
		if err != nil {
			errChan <- err
		}
	}()

	close(ready)

	select {
	case <-signals:
		return listener.Close()
	case err := <-errChan:
		return err
	}
}
