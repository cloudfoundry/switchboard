package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
)

var port = flag.Uint("port", 29996, "port to listen on")

var (
	statusCode int
	hang       bool
)

func HelloServer(w http.ResponseWriter, req *http.Request) {
	if hang {
		select {}
	}

	w.WriteHeader(statusCode)
	switch statusCode {
	case http.StatusOK:
		io.WriteString(w, "synced")
	case http.StatusServiceUnavailable:
		io.WriteString(w, "")
	}
}

func set200Server(w http.ResponseWriter, req *http.Request) {
	statusCode = http.StatusOK
	io.WriteString(w, "will return 200 on /")
}

func set503Server(w http.ResponseWriter, req *http.Request) {
	statusCode = http.StatusServiceUnavailable
	io.WriteString(w, "will return 503 on /")
}

func setHangServer(w http.ResponseWriter, req *http.Request) {
	hang = true
	io.WriteString(w, "will hang on /")
}

func main() {
	flag.Parse()
	fmt.Printf("Healthcheck listening on port %d\n", *port)
	statusCode = http.StatusOK
	hang = false

	http.HandleFunc("/", HelloServer)
	http.HandleFunc("/set200", set200Server)
	http.HandleFunc("/set503", set503Server)
	http.HandleFunc("/setHang", setHangServer)

	err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
