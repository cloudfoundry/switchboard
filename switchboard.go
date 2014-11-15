package switchboard

import (
	"net"

	"github.com/cloudfoundry-incubator/cf-lager"
	"github.com/pivotal-golang/lager"
)

type Switchboard struct {
	logger   lager.Logger
	listener net.Listener
	cluster  Cluster
}

func New(listener net.Listener, cluster Cluster) Switchboard {
	return Switchboard{
		logger:   cf_lager.New("switchboard"),
		listener: listener,
		cluster:  cluster,
	}
}

func (s Switchboard) Run() {
	s.logger.Info("Running switchboard ...")
	upChan, downChan := s.cluster.Start()
	connChan := s.acceptConnections()
	s.servingConnections(connChan, upChan, downChan)
}

// Takes connection channel (connChan) and channels monitoring cluster health (upChan/downChan)
// Immediately closes connections
// (Eventually we would like to send a meaningful error message before closing)
// Defers to servingConnections when message comes on upChan
func (s Switchboard) rejectConnections(connChan <-chan net.Conn, upChan <-chan struct{}, downChan <-chan struct{}) {
	s.logger.Info("Rejecting Connections.")
	for {
		select {
		case conn := <-connChan:
			conn.Close()
		case <-upChan:
			s.servingConnections(connChan, upChan, downChan)
			return
		}
	}
}

// Takes connection channel (connChan) and channels monitoring cluster health (upChan/downChan)
// Routes connections to the backend
// Defers to rejectConnections when message comes on downChan
func (s Switchboard) servingConnections(connChan <-chan net.Conn, upChan <-chan struct{}, downChan <-chan struct{}) {
	s.logger.Info("Serving Connections.")
	for {
		select {
		case conn := <-connChan:
			err := s.cluster.RouteToBackend(conn)
			if err != nil {
				conn.Close()
				s.logger.Error("Error routing to backend", err)
			}
		case <-downChan:
			s.rejectConnections(connChan, upChan, downChan)
			return
		}
	}
}

// Return a channel of client connections.
// The inner go routine listens for incoming connections and puts them on
// the connections channel.  We had to put the accept call into a go
// routine since that call is blocking.
func (s Switchboard) acceptConnections() <-chan net.Conn {
	s.logger.Info("Accepting connections ...")
	c := make(chan net.Conn)
	go func() {
		for {
			clientConn, err := s.listener.Accept()
			if err != nil {
				s.logger.Error("Error accepting client connection", err)
			} else {
				c <- clientConn
			}
		}
	}()
	return c
}
