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

func (pr ProxyRunner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	pr.logger.Info(fmt.Sprintf("Proxy listening on port %d\n", pr.port))
	pr.cluster.Monitor()
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", pr.port))
	if err != nil {
		return err
	}

	go func() {
		for {
			pr.logger.Info("Accepting connections ...")
			clientConn, err := listener.Accept()

			if err != nil {
				pr.logger.Error("Error accepting client connection", err)
			} else {
				pr.logger.Info("Serving Connections.")

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
