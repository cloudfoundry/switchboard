package fakes

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
)

type FakeHealthcheck struct {
	port       uint
	statusCode int
	hang       bool
}

func NewFakeHealthcheck(port uint) *FakeHealthcheck {
	return &FakeHealthcheck{
		port:       port,
		statusCode: http.StatusOK,
		hang:       false,
	}
}

func (fh *FakeHealthcheck) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	fh.hang = false

	errChan := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/", fh.HelloServer)
	mux.HandleFunc("/set200", fh.set200Server)
	mux.HandleFunc("/set503", fh.set503Server)
	mux.HandleFunc("/setHang", fh.setHangServer)

	server := http.Server{
		Handler: mux,
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", fh.port))
	if err != nil {
		return err
	}

	go func() {
		err := server.Serve(listener)
		if err != nil {
			errChan <- err
		}
	}()

	fmt.Printf("Healthcheck listening on port %d\n", fh.port)
	close(ready)

	for {
		select {
		case err := <-errChan:
			return err
		case <-signals:
			listener.Close()
		}
	}
	return nil
}

func (fh *FakeHealthcheck) HelloServer(w http.ResponseWriter, req *http.Request) {
	if fh.hang {
		select {}
	}

	w.WriteHeader(fh.statusCode)
	switch fh.statusCode {
	case http.StatusOK:
		io.WriteString(w, "synced")
	case http.StatusServiceUnavailable:
		io.WriteString(w, "")
	}
}

func (fh *FakeHealthcheck) set200Server(w http.ResponseWriter, req *http.Request) {
	fh.statusCode = http.StatusOK
	io.WriteString(w, "will return 200 on /")
}

func (fh *FakeHealthcheck) set503Server(w http.ResponseWriter, req *http.Request) {
	fh.statusCode = http.StatusServiceUnavailable
	io.WriteString(w, "will return 503 on /")
}

func (fh *FakeHealthcheck) setHangServer(w http.ResponseWriter, req *http.Request) {
	fh.hang = true
	io.WriteString(w, "will hang on /")
}
