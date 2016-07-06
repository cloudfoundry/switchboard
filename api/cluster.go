package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/cloudfoundry-incubator/switchboard/domain"
	"github.com/pivotal-golang/lager"
)

var Cluster = func(cluster domain.Cluster, logger lager.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case "GET":
			writeClusterResponse(cluster, w)
			return
		case "PATCH":
			setAllowTraffic(req, cluster, logger)
			writeClusterResponse(cluster, w)
			return
		default:
			panic("unrecognized method")
		}
	})
}

func writeClusterResponse(cluster domain.Cluster, w http.ResponseWriter) {
	clusterJSON, err := json.Marshal(cluster.AsJSON())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, err = w.Write(clusterJSON)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func setAllowTraffic(req *http.Request, cluster domain.Cluster, logger lager.Logger) {
	logger.Info("Received set allow traffic state")

	err := req.ParseForm()
	if err != nil {
		panic(err)
	}

	enabledStr := req.FormValue("allow_traffic")
	enabled, err := strconv.ParseBool(enabledStr)
	if err != nil {
		panic(err)
	}

	if enabled {
		cluster.EnableTraffic()
	} else {
		cluster.DisableTraffic()
	}
}
