package api

import (
	"crypto/subtle"
	"net/http"

	"github.com/pivotal-cf-experimental/switchboard/domain"
	"github.com/pivotal-golang/lager"
)

func NewHandler(backends domain.Backends, logger lager.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/v0/backends",
		basicAuthHandler(BackendsIndex(backends)))

	return panicRecoveryHandler(mux, logger)
}

func panicRecoveryHandler(next http.Handler, logger lager.Logger) http.Handler {
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
		next.ServeHTTP(rw, req)
	})
}

func basicAuthHandler(next http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		username, password, ok := req.BasicAuth()
		if ok &&
			secureCompare(username, "username") &&
			secureCompare(password, "password") {
			next.ServeHTTP(rw, req)
		} else {
			rw.Header().Set("WWW-Authenticate", "Basic realm=\"Authorization Required\"")
			http.Error(rw, "Not Authorized", http.StatusUnauthorized)
		}
	}
}

func secureCompare(a, b string) bool {
	x := []byte(a)
	y := []byte(b)
	return subtle.ConstantTimeCompare(x, y) == 1
}
