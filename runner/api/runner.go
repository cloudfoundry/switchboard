package api

import (
	"fmt"
	"net/http"

	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"
)

func NewRunner(port uint, handler http.Handler) ifrit.Runner {
	return http_server.New(fmt.Sprintf("0.0.0.0:%d", port), handler)
}
