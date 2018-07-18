package bridge

import (
	"fmt"
	"net"
	"os"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-incubator/switchboard/domain"
)

type Runner struct {
	logger             lager.Logger
	port               uint
	TrafficEnabledChan chan bool
	ActiveBackendChan  chan *domain.Backend
	timeout            time.Duration
}

func NewRunner(
	port uint,
	timeout time.Duration,
	logger lager.Logger,
) Runner {
	backendChan := make(chan *domain.Backend)
	trafficEnabledChan := make(chan bool)

	return Runner{
		logger:             logger,
		ActiveBackendChan:  backendChan,
		TrafficEnabledChan: trafficEnabledChan,
		port:               port,
		timeout:            timeout,
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
		var activeBackend *domain.Backend
		e := make(chan error)
		c := make(chan net.Conn)

		for {
			go blockingAccept(listener, c, e)
			select {
			case <-shutdown:
				return
			case t := <-r.TrafficEnabledChan:
				// ENABLED -> DISABLED
				if trafficEnabled && !t {
					if activeBackend != nil {
						activeBackend.SeverConnections()
					}
				}

				trafficEnabled = t

			case a := <-r.ActiveBackendChan:
				// NEW ACTIVE BACKEND
				if activeBackend != nil {
					activeBackend.SeverConnections()
				}

				activeBackend = a
				if a != nil {
					r.logger.Info("Done severing connections, new active backend:", lager.Data{"backend": a.AsJSON()})
				} else {
					r.logger.Info("Done severing connections, new active backend:", lager.Data{"backend": nil})
				}

			case clientConn := <-c:
				if !trafficEnabled {
					clientConn.Close()
					continue
				}

				go func(clientConn net.Conn, activeBackend *domain.Backend) {
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
				}(clientConn, activeBackend)
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

	time.Sleep(r.timeout)

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
