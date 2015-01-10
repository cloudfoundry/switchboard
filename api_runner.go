package switchboard

import (
	"fmt"
	"io"
	"net"
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
		listener.Close()
		return nil
	case err := <-errChan:
		return err
	}
}
