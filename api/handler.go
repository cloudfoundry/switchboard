package api

import (
	"net/http"

	"github.com/pivotal-cf-experimental/switchboard/api/middleware"
	"github.com/pivotal-cf-experimental/switchboard/domain"
	"github.com/pivotal-golang/lager"
)

func NewHandler(backends domain.Backends, logger lager.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/v0/backends", BackendsIndex(backends))

	return middleware.Chain{
		middleware.NewPanicRecovery(logger),
		middleware.NewBasicAuth(),
	}.Wrap(mux)
}
