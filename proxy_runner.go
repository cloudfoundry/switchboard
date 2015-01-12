package switchboard

import (
	"fmt"
	"net"
	"os"

	"github.com/pivotal-golang/lager"
)

type ProxyRunner struct {
	logger  lager.Logger
	port    uint
	cluster Cluster
}

func NewProxyRunner(cluster Cluster, port uint, logger lager.Logger) ProxyRunner {
	return ProxyRunner{
		logger:  logger,
		port:    port,
		cluster: cluster,
	}
}

func (s ProxyRunner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	s.logger.Info("Running switchboard ...")
	s.cluster.Monitor()
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", s.port))
	if err != nil {
		return err
	}

	go func() {
		for {
			s.logger.Info("Accepting connections ...")
			clientConn, err := listener.Accept()

			if err != nil {
				s.logger.Error("Error accepting client connection", err)
			} else {
				s.logger.Info("Serving Connections.")

				err := s.cluster.RouteToBackend(clientConn)
				if err != nil {
					clientConn.Close()
					s.logger.Error("Error routing to backend", err)
				}
			}
		}
	}()

	close(ready)
	<-signals
	return listener.Close()
}
