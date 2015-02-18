package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/pivotal-golang/lager"
)

type Logger struct {
	logger      lager.Logger
	routePrefix string
}

func NewLogger(logger lager.Logger, routePrefix string) Middleware {
	return Logger{
		logger:      logger,
		routePrefix: routePrefix,
	}
}

func (l Logger) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		loggingResponseWriter := responseWriter{
			rw,
			[]byte{},
			0,
		}
		next.ServeHTTP(&loggingResponseWriter, req)

		if strings.HasPrefix(req.URL.String(), l.routePrefix) {
			requestCopy := *req
			requestCopy.Header["Authorization"] = nil

			response := map[string]interface{}{
				"Header":     loggingResponseWriter.Header(),
				"Body":       string(loggingResponseWriter.body),
				"StatusCode": loggingResponseWriter.statusCode,
			}

			l.logger.Info("", lager.Data{
				"request":  requestCopy,
				"response": response,
			})
		}
	})
}

type responseWriter struct {
	http.ResponseWriter
	body       []byte
	statusCode int
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.Header().Set("Content-Length", strconv.Itoa(len(b)))

	if rw.statusCode == 0 {
		rw.WriteHeader(http.StatusOK)
	}

	size, err := rw.ResponseWriter.Write(b)
	rw.body = b
	return size, err
}

func (rw *responseWriter) WriteHeader(s int) {
	rw.statusCode = s
	rw.ResponseWriter.WriteHeader(s)
}
