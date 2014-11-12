package main

import (
	"flag"
	"fmt"
	"log"
	"net"
)

var port = flag.Uint("port", 19996, "port to listen on")

const (
	CONN_HOST = "localhost"
	CONN_TYPE = "tcp"
)

func main() {
	flag.Parse()

	address := fmt.Sprintf("%s:%d", CONN_HOST, *port)

	// Listen for incoming connections.
	l, err := net.Listen(CONN_TYPE, address)
	if err != nil {
		log.Fatal("Error listening: %s\n", err.Error())
	}

	// Close the listener when the application closes.
	defer l.Close()
	fmt.Printf("Backend listening on port %s\n", address)
	for {
		conn, err := l.Accept()
		defer conn.Close()
		if err != nil {
			log.Fatal("Error accepting: ", err.Error())
		}
		go handleRequest(conn)
	}
}

// Handles incoming requests.
func handleRequest(conn net.Conn) {
	dataCh := make(chan []byte)
	errCh := make(chan error)

	go func(ch chan []byte, eCh chan error) {
		for {
			data := make([]byte, 1024)
			n, err := conn.Read(data)
			fmt.Println("Dummy listener received on connection: " + string(data))
			if err != nil {
				eCh <- err
				return
			}
			ch <- data[:n]
		}
	}(dataCh, errCh)

	for {
		select {
		case data := <-dataCh:
			fmt.Println("Dummy listener writing to connection: Echo: " + string(data))
			conn.Write([]byte(fmt.Sprintf("Echo from port %d: %s", *port, string(data))))
		case err := <-errCh:
			fmt.Println("Error: " + err.Error())
			conn.Close()
			break
		}
	}
}
