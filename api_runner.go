package switchboard

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
)

type APIRunner struct {
	port     uint
	backends Backends
}

func NewAPIRunner(port uint, backends Backends) APIRunner {
	return APIRunner{
		port:     port,
		backends: backends,
	}
}

func (a APIRunner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	http.HandleFunc("/v0/backends", func(w http.ResponseWriter, req *http.Request) {
		backendsJSON, err := json.Marshal(a.backends.AsJSON())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = w.Write(backendsJSON)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", a.port))
	if err != nil {
		return err
	}
	errChan := make(chan error)
	go func() {
		err := http.Serve(listener, nil)
		if err != nil {
			errChan <- err
		}
	}()

	close(ready)

	select {
	case <-signals:
		return listener.Close()
	case err := <-errChan:
		return err
	}
}
