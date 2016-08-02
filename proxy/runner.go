package proxy

import (
	"fmt"
	"net"
	"os"

	"github.com/cloudfoundry-incubator/switchboard/domain"
	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter . Cluster
type Cluster interface {
	Monitor() chan<- interface{}
	RouteToBackend(clientConn net.Conn) error
	AsJSON() domain.ClusterJSON
	EnableTraffic(message string)
	DisableTraffic(message string)
}

type Runner struct {
	logger  lager.Logger
	port    uint
	cluster Cluster
}

func NewRunner(cluster Cluster, port uint, logger lager.Logger) Runner {
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

				err := pr.cluster.RouteToBackend(clientConn)
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
