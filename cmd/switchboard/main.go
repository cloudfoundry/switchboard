package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
)

var (
	port = flag.Uint("port", 3306, "Port to listen on")

	backendIp   = flag.String("backendIp", "", "IP address of backend")
	backendPort = flag.Uint("backendPort", 3306, "Port of backend")
)

func Connect(frontend, backend net.Conn) {
	defer frontend.Close()
	defer backend.Close()

	select {
	case <-safeCopy(frontend, backend):
	case <-safeCopy(backend, frontend):
	}
}

func safeCopy(from, to net.Conn) chan struct{} {
	done := make(chan struct{})
	go func() {
		_, err := io.Copy(from, to)
		if err != nil {
			fmt.Printf("Error copying from 'from' to 'to': %v\n", err.Error())
		} else {
			fmt.Printf("Copying from 'from' to 'to' completed without an error\n")
		}
		close(done)
	}()
	return done
}

func main() {
	flag.Parse()

	l, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	defer l.Close()
	if err != nil {
		fmt.Printf("Error was: %v\n", err.Error())
	}

	fmt.Printf("Proxy started on port %d\n", *port)

	for {
		clientConn, err := l.Accept()
		defer clientConn.Close()
		if err != nil {
			log.Fatal("Error accepting client connection: %v", err)
		}

		addr := fmt.Sprintf("%s:%d", *backendIp, *backendPort)
		backendConn, err := net.Dial("tcp", addr)
		defer backendConn.Close()
		if err != nil {
			log.Fatal("Error opening backend connection: %v", err)
		}

		go Connect(clientConn, backendConn)
	}
}
