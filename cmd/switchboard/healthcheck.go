package main

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

func (h *HttpHealthcheck) Start(errorCallback func()) {
	go func() {
		for {
			resp, err := http.Get(fmt.Sprintf("http://%s:%d", h.ipAddress, h.port))
			if err != nil {
				fmt.Printf("Error dialing healthchecker: %v\n", err.Error())
			} else {
				switch resp.StatusCode {
				case http.StatusServiceUnavailable:
					errorCallback()
				case http.StatusOK:
					fmt.Printf("Healthcheck at %d succeeded\n", h.port)
				}
				time.Sleep(1 * time.Second)
			}
		}
	}()
}
