package switchboard

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

type Healthcheck interface {
	Start(errorCallback func())
}

type HttpHealthcheck struct {
	ipAddress   string
	port        uint
	timeout     time.Duration
	healthyChan chan bool
	errorChan   chan interface{}
}

func NewHttpHealthCheck(ipAddress string, port uint, timeout time.Duration) *HttpHealthcheck {
	return &HttpHealthcheck{
		ipAddress:   ipAddress,
		port:        port,
		timeout:     timeout,
		errorChan:   make(chan interface{}),
		healthyChan: make(chan bool),
	}
}

func (h HttpHealthcheck) getEndpoint() string {
	endpoint := fmt.Sprintf("http://%s:%d", h.ipAddress, h.port)
	return endpoint
}

func (h *HttpHealthcheck) Start(errorCallback func()) {
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
				errorCallback()
			case <-timeout:
				errorCallback()
			}
		}
	}()
}

func (h *HttpHealthcheck) check() {
	resp, err := http.Get(h.getEndpoint())
	if err != nil {
		fmt.Printf("Error dialing healthchecker at %s: %v\n", h.getEndpoint(), err.Error())
		close(h.errorChan)
	} else {
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			log.Printf("Healthcheck at %s succeeded\n", h.getEndpoint())
			h.healthyChan <- true
		} else {
			close(h.errorChan)
		}
	}
}
