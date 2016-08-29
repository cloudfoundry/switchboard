package monitor

import (
	"os"

	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter . Monitor
type Monitor interface {
	Monitor(<-chan interface{})
}

type Runner struct {
	logger  lager.Logger
	monitor Monitor
}

func NewRunner(monitor Monitor, logger lager.Logger) Runner {
	return Runner{
		logger:  logger,
		monitor: monitor,
	}
}

func (pr Runner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	shutdown := make(chan interface{})
	pr.monitor.Monitor(shutdown)

	close(ready)

	signal := <-signals
	pr.logger.Info("Received signal", lager.Data{"signal": signal})
	close(shutdown)

	return nil
}
