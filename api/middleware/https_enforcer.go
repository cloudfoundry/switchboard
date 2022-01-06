package middleware

import (
	"net/http"
	"net/url"
	"strings"
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
		header := req.Header.Get("X-Forwarded-Proto")
		if !h.forceHttps || h.isHttps(header) {
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

func (h httpsEnforcer) isHttps(header string) bool {
	isHttps := true
	for _, v := range strings.Split(header, ",") {
		if strings.TrimSpace(v) != "https" {
			isHttps = false
		}
	}
	return isHttps
}
