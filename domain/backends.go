package domain

import (
	"github.com/cloudfoundry-incubator/switchboard/config"
	"github.com/pivotal-golang/lager"
)

var BackendProvider = NewBackend

func NewBackends(backendConfigs []config.Backend, logger lager.Logger) (backends []*Backend) {
	for _, bc := range backendConfigs {
		backends = append(backends, BackendProvider(
			bc.Name,
			bc.Host,
			bc.Port,
			bc.StatusPort,
			bc.StatusEndpoint,
			logger,
		))
	}

	return backends
}
