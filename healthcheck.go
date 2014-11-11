package switchboard

import (
	"errors"
	"net/http"
	"time"

	"github.com/pivotal-golang/lager"
)

type Healthcheck interface {
	Start(backend Backend)
}

type HttpHealthcheck struct {
	timeout     time.Duration
	healthyChan chan bool
	errorChan   chan interface{}
	logger      lager.Logger
}

func NewHttpHealthCheck(timeout time.Duration, logger lager.Logger) Healthcheck {
	return &HttpHealthcheck{
		timeout:     timeout,
		errorChan:   make(chan interface{}),
		healthyChan: make(chan bool),
		logger:      logger,
	}
}

func (h HttpHealthcheck) Start(backend Backend) {
	go func() {
		healthCheckInterval := time.Tick(h.timeout / 5)
		for _ = range healthCheckInterval {
			h.check(backend.HealthcheckUrl())
		}
	}()

	go func() {
		for {
			timeout := time.After(h.timeout)
			select {
			case <-h.healthyChan:
				timeout = time.After(h.timeout)
			case <-h.errorChan:
				backend.RemoveAndCloseAllBridges()
			case <-timeout:
				backend.RemoveAndCloseAllBridges()
			}
		}
	}()
}

func (h HttpHealthcheck) check(url string) {
	resp, err := http.Get(url)
	if err != nil {
		h.logger.Error("Error dialing healthchecker", err, lager.Data{"endpoint": url})
		h.errorChan <- err
	} else {
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			h.logger.Debug("Healthcheck succeeded", lager.Data{"endpoint": url})
			h.healthyChan <- true
		} else {
			h.logger.Error("Non-200 exit code from healthcheck", errors.New("Non-200 exit code from healthcheck"))
			h.errorChan <- errors.New("Non-200 exit code from healthcheck")
		}
	}
}
