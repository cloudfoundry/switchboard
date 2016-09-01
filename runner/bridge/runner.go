package bridge

import (
	"fmt"
	"net"
	"os"

	"github.com/cloudfoundry-incubator/switchboard/domain"
	"github.com/pivotal-golang/lager"
)

type ActiveBackendRepository interface {
	Active() domain.Backend
}

type Runner struct {
	logger             lager.Logger
	port               uint
	trafficEnabledChan <-chan bool
	activeRepo         ActiveBackendRepository
}

func NewRunner(activeRepo ActiveBackendRepository, trafficEnabledChan <-chan bool, port uint, logger lager.Logger) Runner {
	return Runner{
		logger:             logger,
		activeRepo:         activeRepo,
		trafficEnabledChan: trafficEnabledChan,
		port:               port,
	}
}

func (r Runner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	r.logger.Info(fmt.Sprintf("Proxy listening on port %d", r.port))

	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", r.port))
	if err != nil {
		return err
	}

	shutdown := make(chan interface{})
	go func(shutdown <-chan interface{}, listener net.Listener) {
		trafficEnabled := true
		for {

			e := make(chan error)
			c := make(chan net.Conn)

			go blockingAccept(listener, c, e)
			select {
			case <-shutdown:
				return
			case t := <-r.trafficEnabledChan:
				// ENABLED -> DISABLED
				if trafficEnabled && !t {
					r.activeRepo.Active().SeverConnections()
				}

				trafficEnabled = t
				close(e)
				close(c)
				continue
			case clientConn := <-c:
				if !trafficEnabled {
					clientConn.Close()
					continue
				}

				go func(clientConn net.Conn) {
					activeBackend := r.activeRepo.Active()
					if activeBackend == nil {
						clientConn.Close()
						r.logger.Error("No active backend", err)
						return
					}

					err := activeBackend.Bridge(clientConn)
					if err != nil {
						clientConn.Close()
						r.logger.Error("Error routing to backend", err)
					}
				}(clientConn)
			case err := <-e:
				if err != nil {
					r.logger.Error("Error accepting client connection", err)
					continue
				}

			}
		}
	}(shutdown, listener)

	close(ready)

	signal := <-signals
	r.logger.Info("Received signal", lager.Data{"signal": signal})
	close(shutdown)
	listener.Close()

	r.logger.Info("Proxy runner has exited")
	return nil
}

func blockingAccept(l net.Listener, c chan<- net.Conn, e chan<- error) {
	clientConn, err := l.Accept()

	if err != nil {
		e <- err
		return
	}
	c <- clientConn
}
