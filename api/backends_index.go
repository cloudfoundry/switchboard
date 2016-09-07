package api

import (
	"encoding/json"
	"net/http"

	"github.com/cloudfoundry-incubator/switchboard/domain"
)

var BackendsIndex = func(backends []*domain.Backend, cluster ClusterManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		backendsJSON, err := json.Marshal(Backends(backends).AsV0JSON(cluster))

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, err = w.Write(backendsJSON)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

type Backends []*domain.Backend

type V0BackendResponse struct {
	Host                string `json:"host"`
	Port                uint   `json:"port"`
	Healthy             bool   `json:"healthy"`
	Name                string `json:"name"`
	CurrentSessionCount uint   `json:"currentSessionCount"`
	Active              bool   `json:"active"`         // For Backwards Compatibility
	TrafficEnabled      bool   `json:"trafficEnabled"` // For Backwards Compatibility
}

func (bs Backends) AsV0JSON(cluster ClusterManager) (json []V0BackendResponse) {
	cj := cluster.AsJSON()
	activeBackend := cj.ActiveBackend

	for _, b := range bs {
		j := b.AsJSON()

		json = append(json, V0BackendResponse{
			Host:                j.Host,
			Port:                j.Port,
			Healthy:             j.Healthy,
			Name:                j.Name,
			CurrentSessionCount: j.CurrentSessionCount,
			Active:              activeBackend != nil && j.Host == activeBackend.Host,
			TrafficEnabled:      cj.TrafficEnabled,
		})
	}

	return json
}
