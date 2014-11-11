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

func (s *Switchboard) Run() {
	s.cluster.StartHealthchecks()
	for {
		clientConn, err := s.listener.Accept()
		if err != nil {
			s.logger.Error("Error accepting client connection", err)
		} else {
			err := s.cluster.RouteToBackend(clientConn)
			if err != nil {
				s.logger.Error("Error routing to backend", err)
			}
		}
	}
}
