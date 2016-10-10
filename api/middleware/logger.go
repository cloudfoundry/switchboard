package middleware

import (
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"code.cloudfoundry.org/lager"
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

			reqCopy := *req
			reqCopy.Header["Authorization"] = nil

			var reqBody string
			if reqCopy.Body != nil {
				bodyBytes, err := ioutil.ReadAll(reqCopy.Body)
				if err != nil {
					l.logger.Error("Could not read response body", err)
					reqBody = ""
				} else {
					reqBody = string(bodyBytes)
				}
			}

			requestData := map[string]interface{}{
				"Header":     reqCopy.Header,
				"Body":       reqBody,
				"URL":        reqCopy.URL,
				"Host":       reqCopy.Host,
				"RemoteAddr": reqCopy.RemoteAddr,
			}

			responseData := map[string]interface{}{
				"Header":     loggingResponseWriter.Header(),
				"Body":       string(loggingResponseWriter.body),
				"StatusCode": loggingResponseWriter.statusCode,
			}

			l.logger.Debug("", lager.Data{
				"request":  requestData,
				"response": responseData,
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
