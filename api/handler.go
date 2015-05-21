package api

import (
	"net/http"

	"github.com/cloudfoundry-incubator/switchboard/api/middleware"
	"github.com/cloudfoundry-incubator/switchboard/config"
	"github.com/cloudfoundry-incubator/switchboard/domain"
	"github.com/pivotal-golang/lager"
)

func NewHandler(backends domain.Backends, logger lager.Logger, apiConfig config.API, staticDir string) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/", http.FileServer(http.Dir(staticDir)))

	mux.Handle("/v0/backends", BackendsIndex(backends))

	return middleware.Chain{
		middleware.NewPanicRecovery(logger),
		middleware.NewLogger(logger, "/v0"),
		middleware.NewHttpsEnforcer(apiConfig.ForceHttps),
		middleware.NewBasicAuth(apiConfig.Username, apiConfig.Password),
	}.Wrap(mux)
}
