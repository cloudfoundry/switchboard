package health

import (
	"fmt"
	"net"
	"os"

	"github.com/pivotal-golang/lager"
)

type Runner struct {
	logger lager.Logger
	port   uint
}

func NewRunner(port uint, logger lager.Logger) Runner {
	return Runner{
		logger: logger,
		port:   port,
	}
}

func (a Runner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {

	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", a.port))
	if err != nil {
		a.logger.Error("Error on Listen", err)
		return err
	} else {
		a.logger.Info(fmt.Sprintf("Proxy health listening on port %d", a.port))
	}

	exitChan := make(chan struct{})
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-exitChan:
					return
				default:
					a.logger.Error(fmt.Sprintf("Accepting TCP connection on health port %d", a.port), err)
				}
			}
			err = conn.Close()
			if err != nil {
				a.logger.Error(fmt.Sprintf("Closing TCP connection from %s on health port %d", conn.RemoteAddr(), a.port), err)
			}
		}
	}()

	close(ready)

	signal := <-signals
	a.logger.Info("Received signal", lager.Data{"signal": signal})

	// gracefully exit the goroutine - listener.Close causes Accept to error
	err = listener.Close()
	if err != nil {
		a.logger.Error("Closed Health Runner", err)
	}
	close(exitChan)
	return err
}
