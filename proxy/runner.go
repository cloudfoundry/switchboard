package proxy

import (
	"fmt"
	"net"
	"os"

	"github.com/cloudfoundry-incubator/switchboard/domain"
	"github.com/pivotal-golang/lager"
)

type Runner struct {
	logger  lager.Logger
	port    uint
	cluster domain.Cluster
}

func NewRunner(cluster domain.Cluster, port uint, logger lager.Logger) Runner {
	return Runner{
		logger:  logger,
		port:    port,
		cluster: cluster,
	}
}

func (pr Runner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	pr.logger.Info(fmt.Sprintf("Proxy listening on port %d", pr.port))
	pr.cluster.Monitor()
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", pr.port))
	if err != nil {
		return err
	}

	go func() {
		for {
			clientConn, err := listener.Accept()

			if err != nil {
				pr.logger.Error("Error accepting client connection", err)
			} else {

				err := pr.cluster.RouteToBackend(clientConn)
				if err != nil {
					clientConn.Close()
					pr.logger.Error("Error routing to backend", err)
				}
			}
		}
	}()

	close(ready)
	<-signals
	return listener.Close()
}
