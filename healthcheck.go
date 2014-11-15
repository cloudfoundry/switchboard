package switchboard

import (
	"net/http"
	"time"

	"github.com/cloudfoundry-incubator/cf-lager"
	"github.com/pivotal-golang/lager"
)

type Healthcheck interface {
	Start(backend Backend) (<-chan Backend, <-chan Backend)
}

type healthcheck struct {
	timeout time.Duration
	logger  lager.Logger
}

func NewHealthcheck(timeout time.Duration) Healthcheck {
	return &healthcheck{
		timeout: timeout,
		logger:  cf_lager.New("healthcheck"),
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

	resp, err := h.getWithTimeout(url, h.timeout)

	if err != nil {
		h.logger.Error("Error dialing healthchecker", err, lager.Data{"endpoint": url})
		h.nonBlockingWrite(unhealthyChan, backend)
	} else {
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			h.logger.Debug("Healthcheck succeeded", lager.Data{"endpoint": url})
			h.nonBlockingWrite(healthyChan, backend)
		} else {
			h.logger.Debug("Non-200 exit code from healthcheck", lager.Data{"status_code": resp.StatusCode, "endpoint": url})
			h.nonBlockingWrite(unhealthyChan, backend)
		}
	}
}

func (h healthcheck) getWithTimeout(url string, timeout time.Duration) (*http.Response, error) {
	client := http.Client{
		Timeout: timeout,
	}
	return client.Get(url)
}

func (h healthcheck) nonBlockingWrite(channel chan Backend, backend Backend) {
	select {
	case channel <- backend:
	default:
	}
}
