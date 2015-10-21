package health

import (
	"fmt"
	"net"
	"os"

	"github.com/pivotal-golang/lager"
)

type Runner struct {
	logger lager.Logger
	port   uint
}

func NewRunner(port uint, logger lager.Logger) Runner {
	return Runner{
		logger: logger,
		port:   port,
	}
}

func (a Runner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {

	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", a.port))
	if err != nil {
		a.logger.Error("Error on Listen", err)
		return err
	} else {
		a.logger.Info(fmt.Sprintf("Proxy health listening on port %d", a.port))
	}

	shutdown := make(chan struct{})
	go func() {
		for {
			conn, err := listener.Accept()

			select {
			case <-shutdown:
				return
			default:
				//continue
			}

			if err != nil {
				a.logger.Error("Error accepting health check connection", err)
			} else {
				err = conn.Close()
				if err != nil {
					a.logger.Error("Error closing health check connection", err)
				}
			}
		}
	}()

	close(ready)

	signal := <-signals
	a.logger.Info("Received signal", lager.Data{"signal": signal})
	close(shutdown)
	listener.Close()

	a.logger.Info("Health runner has exited")
	return nil
}
