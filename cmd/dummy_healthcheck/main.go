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
)

func HelloServer(w http.ResponseWriter, req *http.Request) {
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

func main() {
	flag.Parse()
	fmt.Printf("Healthcheck listening on port %d\n", *port)
	statusCode = http.StatusOK
	http.HandleFunc("/", HelloServer)
	http.HandleFunc("/set200", set200Server)
	http.HandleFunc("/set503", set503Server)
	err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
