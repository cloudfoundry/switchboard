package bridge

import (
	"errors"
	"net"
	"github.com/cloudfoundry-incubator/switchboard/domain"
)

//go:generate counterfeiter . ActiveBackendRepository
type ActiveBackendRepository interface{
	Active() domain.Backend
}

type ClusterRouter struct {
	backends ActiveBackendRepository
}

func NewClusterRouter(backends ActiveBackendRepository) *ClusterRouter {
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
