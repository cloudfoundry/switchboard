package bridge

import (
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/pivotal-golang/lager"
)

type Router interface {
	RouteToBackend(clientConn net.Conn) error
}

type Runner struct {
	logger             lager.Logger
	port               uint
	router             Router
	trafficEnabledChan <-chan bool
	activeRepo         ActiveBackendRepository
}

func NewRunner(router Router, activeRepo ActiveBackendRepository, trafficEnabledChan <-chan bool, port uint, logger lager.Logger) Runner {
	return Runner{
		logger:             logger,
		activeRepo:         activeRepo,
		trafficEnabledChan: trafficEnabledChan,
		port:               port,
		router:             router,
	}
}

func (r Runner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	r.logger.Info(fmt.Sprintf("Proxy listening on port %d", r.port))

	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", r.port))
	if err != nil {
		return err
	}

	shutdown := make(chan interface{})
	trafficEnabled := true
	var m sync.RWMutex

	go func(shutdown <-chan interface{}) {
		for {
			select {
			case <-shutdown:
				return

			case t := <-r.trafficEnabledChan:
				m.RLock()
				if trafficEnabled && !t {
					r.activeRepo.Active().SeverConnections()
				}
				m.RUnlock()

				m.Lock()
				trafficEnabled = t
				m.Unlock()
			}
		}

	}(shutdown)

	go func(shutdown <-chan interface{}) {
		for {
			select {
			case <-shutdown:
				return

			default:
				clientConn, err := listener.Accept()
				if err != nil {
					r.logger.Error("Error accepting client connection", err)
					continue
				}

				m.RLock()
				if !trafficEnabled {
					m.RUnlock()
					clientConn.Close()
					continue
				}
				m.RUnlock()

				go func(clientConn net.Conn) {
					err := r.router.RouteToBackend(clientConn)
					if err != nil {
						clientConn.Close()
						r.logger.Error("Error routing to backend", err)
					}
				}(clientConn)
			}
		}
	}(shutdown)

	close(ready)

	signal := <-signals
	r.logger.Info("Received signal", lager.Data{"signal": signal})
	close(shutdown)
	listener.Close()

	r.logger.Info("Proxy runner has exited")
	return nil
}
