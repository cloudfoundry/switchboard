package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
)

var (
	pidfile = flag.String("pidfile", "", "The location for the pidfile")
	port    = flag.Uint("port", 3306, "Port to listen on")

	backendIp       = flag.String("backendIp", "", "IP address of backend")
	backendPort     = flag.Uint("backendPort", 3306, "Port of backend")
	healthcheckPort = flag.Uint("healthcheckPort", 9200, "Port for healthcheck endpoints")
)

func acceptClientConnection(l net.Listener) net.Conn {
	clientConn, err := l.Accept()
	if err != nil {
		log.Fatal("Error accepting client connection: %v", err)
	}
	return clientConn
}

func main() {
	flag.Parse()

	l, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	defer l.Close()
	if err != nil {
		log.Fatal(fmt.Sprintf("Error listening on port %d: %v\n", *port, err.Error()))
	}

	err = ioutil.WriteFile(*pidfile, []byte(strconv.Itoa(os.Getpid())), 0644)
	if err != nil {
		log.Fatal(fmt.Sprintf("Cannot write pid to file: %s", *pidfile))
	}

	fmt.Printf("Proxy started on port %d\n", *port)

	backend := NewBackend("backend1", *backendIp, *backendPort)

	healthcheck := NewHttpHealthCheck(*backendIp, *healthcheckPort)
	healthcheck.Start(backend.RemoveAllBridges)

	for {
		clientConn := acceptClientConnection(l)
		defer clientConn.Close()

		backendConn, err := backend.Dial()
		if err != nil {
			log.Fatal("Error connection to backend: %s", err.Error())
		}
		defer backendConn.Close()

		bridge := NewBridge(clientConn, backendConn)
		backend.AddBridge(bridge)

		go func() {
			bridge.Connect()
			backend.RemoveBridge(bridge)
		}()
	}
}
