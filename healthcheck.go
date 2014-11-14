package switchboard

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/pivotal-golang/lager"
)

type Healthcheck interface {
	Start(backend Backend) (<-chan Backend, <-chan Backend)
}

type healthcheck struct {
	timeout time.Duration
	logger  lager.Logger
}

func NewHealthcheck(timeout time.Duration, logger lager.Logger) Healthcheck {
	return &healthcheck{
		timeout: timeout,
		logger:  logger,
	}
}

func (h healthcheck) Start(backend Backend) (<-chan Backend, <-chan Backend) {
	healthyChan := make(chan Backend)
	unhealthyChan := make(chan Backend)

	go func() {
		healthCheckInterval := time.Tick(h.timeout / 5)
		for _ = range healthCheckInterval {
			h.check(backend, healthyChan, unhealthyChan)
		}
	}()

	// TODO: discuss the purpose and implementation of timeout
	go func() {
		for {
			timeout := time.After(h.timeout)
			select {
			case <-healthyChan:
			case <-unhealthyChan:
			case <-timeout:
				h.logger.Debug(fmt.Sprintf("Healthchecker for backend `%s` timed out", backend.HealthcheckUrl()))
				unhealthyChan <- backend
			}
		}
	}()

	return healthyChan, unhealthyChan
}

func (h healthcheck) check(backend Backend, healthyChan, unhealthyChan chan Backend) {
	url := backend.HealthcheckUrl()
	resp, err := http.Get(url)

	if err != nil {
		h.logger.Error("Error dialing healthchecker", err, lager.Data{"endpoint": url})
		nonBlockingWrite(unhealthyChan, backend)
	} else {
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			h.logger.Debug("Healthcheck succeeded", lager.Data{"endpoint": url})
			nonBlockingWrite(healthyChan, backend)
		} else {
			h.logger.Error("Non-200 exit code from healthcheck", errors.New("Non-200 exit code from healthcheck"))
			nonBlockingWrite(unhealthyChan, backend)
		}
	}
}

func nonBlockingWrite(channel chan Backend, backend Backend) {
	select {
	case channel <- backend:
	default:
	}
}
