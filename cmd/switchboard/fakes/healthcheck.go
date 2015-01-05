package fakes

import (
	"fmt"
	"io"
	"log"
	"net/http"
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

func (fh *FakeHealthcheck) Start() {
	fmt.Printf("Healthcheck listening on port %d\n", fh.port)
	fh.hang = false

	http.HandleFunc("/", fh.HelloServer)
	http.HandleFunc("/set200", fh.set200Server)
	http.HandleFunc("/set503", fh.set503Server)
	http.HandleFunc("/setHang", fh.setHangServer)

	err := http.ListenAndServe(fmt.Sprintf(":%d", fh.port), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
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
