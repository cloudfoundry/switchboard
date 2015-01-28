package api

import (
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/pivotal-golang/lager"
)

type Runner struct {
	logger  lager.Logger
	port    uint
	handler http.Handler
}

func NewRunner(port uint, handler http.Handler, logger lager.Logger) Runner {
	return Runner{
		logger:  logger,
		port:    port,
		handler: handler,
	}
}

func (a Runner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", a.port))
	if err != nil {
		return err
	} else {
		a.logger.Info(fmt.Sprintf("Proxy api listening on port %d", a.port))
	}

	errChan := make(chan error)
	go func() {
		err := http.Serve(listener, a.handler)
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
