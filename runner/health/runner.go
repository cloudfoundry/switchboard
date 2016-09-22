package health

import (
	"fmt"

	"net/http"

	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"
)

func NewRunner(port uint) ifrit.Runner {
	return http_server.New(fmt.Sprintf("0.0.0.0:%d", port), http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(200)
	}))
}
