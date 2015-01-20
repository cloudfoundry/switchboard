package api

import (
	"net/http"

	"github.com/pivotal-cf-experimental/switchboard/domain"
)

func NewHandler(backends domain.Backends) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v0/backends", backendsIndex(backends))
	return mux
}
