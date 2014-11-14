package switchboard

import (
	"errors"
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

	return healthyChan, unhealthyChan
}

func (h healthcheck) check(backend Backend, healthyChan, unhealthyChan chan Backend) {
	url := backend.HealthcheckUrl()

	resp, err := getWithTimeout(url, h.timeout)

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

func getWithTimeout(url string, timeout time.Duration) (*http.Response, error) {
	client := http.Client{
		Timeout: timeout,
	}
	return client.Get(url)
}

func nonBlockingWrite(channel chan Backend, backend Backend) {
	select {
	case channel <- backend:
	default:
	}
}
