package switchboard

import (
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

	client := http.Client{
		Timeout: h.timeout,
	}

	resp, err := client.Get(url)

	if err != nil {
		h.logger.Error("Error dialing healthchecker", err, lager.Data{"endpoint": url})
		select {
		case unhealthyChan <- backend:
		default:
		}
	} else {
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			h.logger.Debug("Healthcheck succeeded", lager.Data{"endpoint": url})
			select {
			case healthyChan <- backend:
			default:
			}
		} else {
			h.logger.Debug("Non-200 exit code from healthcheck", lager.Data{"status_code": resp.StatusCode, "endpoint": url})
			select {
			case unhealthyChan <- backend:
			default:
			}
		}
	}
}
