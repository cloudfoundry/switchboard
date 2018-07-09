package api

import (
	"net/http"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-incubator/switchboard/api/middleware"
	"github.com/cloudfoundry-incubator/switchboard/config"
	"github.com/cloudfoundry-incubator/switchboard/domain"
)

//go:generate counterfeiter -o apifakes/fake_response_writer.go /usr/local/opt/go/libexec/src/net/http/server.go ResponseWriter
func NewHandler(
	clusterManager ClusterManager,
	backends []*domain.Backend,
	logger lager.Logger,
	apiConfig config.API,
	staticDir string,
) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/", http.FileServer(http.Dir(staticDir)))

	mux.Handle("/v0/backends", BackendsIndex(backends, clusterManager))
	mux.Handle("/v0/cluster", ClusterEndpoint(clusterManager, logger))

	return middleware.Chain{
		middleware.NewPanicRecovery(logger),
		middleware.NewLogger(logger, "/v0"),
		middleware.NewHttpsEnforcer(apiConfig.ForceHttps),
		middleware.NewBasicAuth(apiConfig.Username, apiConfig.Password),
	}.Wrap(mux)
}
