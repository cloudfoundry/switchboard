package switchboard

import (
	"fmt"
	"net/http"
	"time"
)

type Healthcheck interface {
	Start(errorCallback func())
}

type HttpHealthcheck struct {
	ipAddress string
	port      uint
}

func NewHttpHealthCheck(ipAddress string, port uint) *HttpHealthcheck {
	return &HttpHealthcheck{
		ipAddress: ipAddress,
		port:      port,
	}
}

func (h HttpHealthcheck) getEndpoint() string {
	endpoint := fmt.Sprintf("http://%s:%d", h.ipAddress, h.port)
	return endpoint
}

func (h *HttpHealthcheck) Start(errorCallback func()) {
	go func() {
		for {
			resp, err := http.Get(h.getEndpoint())
			if err != nil {
				fmt.Printf("Error dialing healthchecker at %s: %v\n", h.getEndpoint(), err.Error())
			} else {
				switch resp.StatusCode {
				case http.StatusServiceUnavailable:
					errorCallback()
				case http.StatusOK:
					fmt.Printf("Healthcheck at %s succeeded\n", h.getEndpoint())
				}
				resp.Body.Close()
			}
			time.Sleep(1 * time.Second)
		}
	}()
}
