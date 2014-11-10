package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-lager"
	. "github.com/pivotal-cf-experimental/switchboard"
	"github.com/pivotal-golang/lager"
)

var (
	pidfile = flag.String("pidfile", "", "The location for the pidfile")
	port    = flag.Uint("port", 3306, "Port to listen on")

	backendIPsFlag       = flag.String("backendIPs", "", "Comma-separated list of backend IP addresses")
	backendPortsFlag     = flag.String("backendPorts", "3306", "Comma-separated list of backend ports")
	healthcheckPortsFlag = flag.String("healthcheckPorts", "9200", "Comma-separated list of healthcheck ports")
	healthcheckTimeout   = flag.Duration("healthcheckTimeout", 5*time.Second, "Timeout for healthcheck")

	backendIPs                     []string
	backendPorts, healthcheckPorts []uint
	logger                         lager.Logger
)

func main() {
	flag.Parse()

	logger = cf_lager.New("switchboard")
	logger.Info("Logging for the switchbord")

	fmt.Println("printing")

	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", *port))
	if err != nil {
		logger.Fatal("Error listening on port.", err, lager.Data{"port": *port})
	}
	defer listener.Close()

	err = ioutil.WriteFile(*pidfile, []byte(strconv.Itoa(os.Getpid())), 0644)
	if err != nil {
		logger.Fatal("Cannot write pid to file", err, lager.Data{"pidfile": *pidfile})
	}

	backendIPs = strings.Split(*backendIPsFlag, ",")

	backendPorts, err = stringsToUints(strings.Split(*backendPortsFlag, ","))
	if err != nil {
		log.Fatal(fmt.Sprintf("Error parsing backendPorts: %v", err))
	}

	healthcheckPorts, err = stringsToUints(strings.Split(*healthcheckPortsFlag, ","))
	if err != nil {
		log.Fatal(fmt.Sprintf("Error parsing healthcheckPorts: %v", err))
	}

	replicatePorts()

	logger.Info(fmt.Sprintf("Proxy started on port %d\n", *port))
	logger.Info(fmt.Sprintf("Backend ipAddress: %s\n", backendIPs[0]))
	logger.Info(fmt.Sprintf("Backend port: %d\n", backendPorts[0]))
	logger.Info(fmt.Sprintf("Healthcheck port: %d\n", healthcheckPorts[0]))

	backends := NewBackends(
		backendIPs,
		backendPorts,
		healthcheckPorts,
		*healthcheckTimeout,
		logger,
	)

	switchboard := New(listener, backends, logger)
	switchboard.Run()
}

func stringsToUints(s []string) ([]uint, error) {
	dest_slice := make([]uint, len(s))
	for i, val := range s {
		intVal, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return nil, err
		}
		dest_slice[i] = uint(intVal)
	}
	return dest_slice, nil
}

func replicatePorts() {
	if len(backendPorts) != len(backendIPs) {
		port := backendPorts[0]
		backendPorts = make([]uint, len(backendIPs))
		for i, _ := range backendIPs {
			backendPorts[i] = port
		}
	}
	if len(healthcheckPorts) != len(backendIPs) {
		port := healthcheckPorts[0]
		healthcheckPorts = make([]uint, len(backendIPs))
		for i, _ := range backendIPs {
			healthcheckPorts[i] = port
		}
	}
}
