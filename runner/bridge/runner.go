package bridge

import (
	"fmt"
	"net"
	"os"

	"github.com/pivotal-golang/lager"
)

type Router interface {
	RouteToBackend(clientConn net.Conn) error
}

type Runner struct {
	logger  lager.Logger
	port    uint
	router Router
}

func NewRunner(router Router, port uint, logger lager.Logger) Runner {
	return Runner{
		logger:  logger,
		port:    port,
		router: router,
	}
}

func (pr Runner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	pr.logger.Info(fmt.Sprintf("Proxy listening on port %d", pr.port))
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", pr.port))
	if err != nil {
		return err
	}

	shutdown := make(chan interface{})
	go func() {
		for {

			clientConn, err := listener.Accept()

			select {
			case <-shutdown:
				return
			default:
				//continue
			}

			if err != nil {
				pr.logger.Error("Error accepting client connection", err)
			} else {

				err := pr.router.RouteToBackend(clientConn)
				if err != nil {
					clientConn.Close()
					pr.logger.Error("Error routing to backend", err)
				}
			}
		}
	}()

	close(ready)

	signal := <-signals
	pr.logger.Info("Received signal", lager.Data{"signal": signal})
	close(shutdown)
	listener.Close()

	pr.logger.Info("Proxy runner has exited")
	return nil
}
