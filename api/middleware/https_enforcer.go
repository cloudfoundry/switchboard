package middleware

import (
	"net/http"
	"net/url"
)

type httpsEnforcer struct {
	forceHttps bool
}

func NewHttpsEnforcer(forceHttps bool) Middleware {
	return httpsEnforcer{
		forceHttps: forceHttps,
	}
}

func (h httpsEnforcer) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if !h.forceHttps || req.Header.Get("X-Forwarded-Proto") == "https" {
			next.ServeHTTP(rw, req)
			return
		}

		redirectTo, err := url.Parse(req.URL.String())
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte(http.StatusText(http.StatusBadRequest)))
			return
		}

		redirectTo.Host = req.Host
		redirectTo.Scheme = "https"

		http.Redirect(rw, req, redirectTo.String(), http.StatusFound)
	})
}
