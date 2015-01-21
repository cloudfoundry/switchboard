package api

import (
	"net/http"

	"github.com/pivotal-cf-experimental/switchboard/domain"
	"github.com/pivotal-golang/lager"
)

func NewHandler(backends domain.Backends, logger lager.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v0/backends", BackendsIndex(backends))

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		defer func() {
			if panicInfo := recover(); panicInfo != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				logger.Error("Panic while serving request", nil, lager.Data{
					"request":   req,
					"panicInfo": panicInfo,
				})
			}
		}()
		mux.ServeHTTP(rw, req)
	})
}
