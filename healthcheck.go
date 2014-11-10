package switchboard

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/pivotal-golang/lager"
)

type Healthcheck interface {
	Start(backend Backend)
}

type HttpHealthcheck struct {
	ipAddress   string
	port        uint
	timeout     time.Duration
	healthyChan chan bool
	errorChan   chan interface{}
	logger      lager.Logger
}

func NewHttpHealthCheck(ipAddress string, port uint, timeout time.Duration, logger lager.Logger) *HttpHealthcheck {
	return &HttpHealthcheck{
		ipAddress:   ipAddress,
		port:        port,
		timeout:     timeout,
		errorChan:   make(chan interface{}),
		healthyChan: make(chan bool),
		logger:      logger,
	}
}

func (h HttpHealthcheck) getEndpoint() string {
	endpoint := fmt.Sprintf("http://%s:%d", h.ipAddress, h.port)
	return endpoint
}

func (h *HttpHealthcheck) Start(backend Backend) {
	go func() {
		for {
			h.check()
			time.Sleep(h.timeout / 5)
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

func (h *HttpHealthcheck) check() {
	resp, err := http.Get(h.getEndpoint())
	if err != nil {
		h.logger.Error("Error dialing healthchecker", err, lager.Data{"endpoint": h.getEndpoint()})
		h.errorChan <- err
	} else {
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			h.logger.Debug("Healthcheck succeeded", lager.Data{"endpoint": h.getEndpoint()})
			h.healthyChan <- true
		} else {
			h.logger.Error("Non-200 exit code from healthcheck", errors.New("Non-200 exit code from healthcheck"))
			h.errorChan <- errors.New("Non-200 exit code from healthcheck")
		}
	}
}
