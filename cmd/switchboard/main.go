package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/cloudfoundry-incubator/cf-lager"
	. "github.com/pivotal-cf-experimental/switchboard"
	"github.com/pivotal-golang/lager"
)

var (
	pidfile = flag.String("pidfile", "", "The location for the pidfile")
	port    = flag.Uint("port", 3306, "Port to listen on")

	backendIp          = flag.String("backendIp", "", "IP address of backend")
	backendPort        = flag.Uint("backendPort", 3306, "Port of backend")
	healthcheckPort    = flag.Uint("healthcheckPort", 9200, "Port for healthcheck endpoints")
	healthcheckTimeout = flag.Duration("healthcheckTimeout", 5*time.Second, "Timeout for healthcheck")
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

	logger := cf_lager.New("switchboard")
	logger.Info("Logging for the switchbord")

	fmt.Println("printing")

	l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", *port))
	if err != nil {
		logger.Fatal("Error listening on port.", err, lager.Data{"port": *port})
	}
	defer l.Close()

	err = ioutil.WriteFile(*pidfile, []byte(strconv.Itoa(os.Getpid())), 0644)
	if err != nil {
		logger.Fatal("Cannot write pid to file", err, lager.Data{"pidfile": *pidfile})
	}

	logger.Info(fmt.Sprintf("Proxy started on port %d\n", *port))
	logger.Info(fmt.Sprintf("Backend ipAddress: %s\n", *backendIp))
	logger.Info(fmt.Sprintf("Backend port: %d\n", *port))
	logger.Info(fmt.Sprintf("Healthcheck port: %d\n", *healthcheckPort))

	backend := NewBackend("backend1", *backendIp, *backendPort)

	healthcheck := NewHttpHealthCheck(*backendIp, *healthcheckPort, *healthcheckTimeout, logger)
	healthcheck.Start(backend.RemoveAndCloseAllBridges)

	for {
		clientConn := acceptClientConnection(l)
		defer clientConn.Close()

		backendConn, err := backend.Dial()
		if err != nil {
			logger.Fatal("Error connection to backend.", err)
		}
		defer backendConn.Close()

		bridge := NewConnectionBridge(clientConn, backendConn, logger)
		backend.AddBridge(bridge)

		go func() {
			bridge.Connect()
			backend.RemoveBridge(bridge)
		}()
	}
}
