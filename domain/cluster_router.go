package domain

import (
	"errors"
	"net"
)

type ClusterRouter struct {
	backends Backends
}

func NewClusterRouter(backends Backends) *ClusterRouter {
	return &ClusterRouter{
		backends: backends,
	}
}

func (c *ClusterRouter) RouteToBackend(clientConn net.Conn) error {
	activeBackend := c.backends.Active()
	if activeBackend == nil {
		return errors.New("No active Backend")
	}
	return activeBackend.Bridge(clientConn)
}
