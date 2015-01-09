package switchboard

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

type APIRunner struct {
	port uint
}

func NewAPIRunner(port uint) APIRunner {
	return APIRunner{
		port: port,
	}
}

func (a APIRunner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, "{}")
	})
	close(ready)

	err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", a.port), nil)
	if err != nil {
		return err
	}
	return nil
}
