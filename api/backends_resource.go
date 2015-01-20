package api

import (
	"encoding/json"
	"net/http"

	"github.com/pivotal-cf-experimental/switchboard/domain"
)

func backendsIndex(backends domain.Backends) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		backendsJSON, err := json.Marshal(backends.AsJSON())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = w.Write(backendsJSON)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
