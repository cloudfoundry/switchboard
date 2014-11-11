package switchboard

import (
	"net"

	"github.com/pivotal-golang/lager"
)

type Switchboard struct {
	logger   lager.Logger
	listener net.Listener
	cluster  Cluster
}

func New(listener net.Listener, cluster Cluster, logger lager.Logger) Switchboard {
	return Switchboard{
		logger:   logger,
		listener: listener,
		cluster:  cluster,
	}
}

func (bm *Switchboard) Run() {
	bm.cluster.StartHealthchecks()
	for {
		clientConn, err := bm.listener.Accept()
		if err != nil {
			bm.logger.Error("Error accepting client connection", err)
		} else {
			err := bm.cluster.RouteToBackend(clientConn)
			if err != nil {
				bm.logger.Error("Error routing to backend", err)
			}
		}
	}
}
