package api

import (
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"strconv"

	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter . ClusterManager
type ClusterManager interface {
	AsJSON() ClusterJSON
	EnableTraffic(string)
	DisableTraffic(string)
}

var Cluster = func(cluster ClusterManager, logger lager.Logger) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case "GET":
			writeClusterResponse(w, cluster)
			return
		case "PATCH":
			handleUpdate(w, req, cluster, logger)
			writeClusterResponse(w, cluster)
			return
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func writeClusterResponse(w http.ResponseWriter, cluster ClusterManager) {
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

func handleUpdate(
	w http.ResponseWriter,
	req *http.Request,
	cluster ClusterManager,
	logger lager.Logger,
) {
	logger.Debug("API /cluster update")

	err := req.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	dumpBody := true
	b, err := httputil.DumpRequest(req, dumpBody)
	if err != nil {
		http.Error(w, "Failed to dump http body", http.StatusInternalServerError)
		return
	}

	logger.Debug("API /cluster req", lager.Data{"dump": string(b)})
	logger.Debug("API /cluster req form", lager.Data{"form": req.Form})

	enabledStr := req.Form.Get("trafficEnabled")
	enabled, err := strconv.ParseBool(enabledStr)
	if err != nil {
		http.Error(w, "Failed to parse trafficEnabled", http.StatusBadRequest)
		return
	}

	if enabled {
		message := req.Form.Get("message")
		cluster.EnableTraffic(message)
	} else {
		message := req.Form.Get("message")
		if message == "" {
			http.Error(w, "message must not be empty", http.StatusBadRequest)
			return
		}
		cluster.DisableTraffic(message)
	}
}
