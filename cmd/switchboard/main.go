package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/cloudfoundry-incubator/cf-lager"
	"github.com/fraenkel/candiedyaml"
	"github.com/pivotal-cf-experimental/switchboard"
	"github.com/pivotal-golang/lager"
)

var (
	config = flag.String("config", "", "Path to config file")

	backendIPs                     []string
	backendPorts, healthcheckPorts []uint
	logger                         lager.Logger
)

type Config struct {
	Port                   uint
	Pidfile                string
	Backends               []Backend
	HealthcheckTimeoutInMS uint
}

type Backend struct {
	BackendIP       string
	BackendPort     uint
	HealthcheckPort uint
}

func main() {
	flag.Parse()

	logger = cf_lager.New("main")

	file, err := os.Open(*config)
	if err != nil {
		logger.Fatal("Config file does not exist:", err, lager.Data{"config": *config})
	}

	config := new(Config)
	decoder := candiedyaml.NewDecoder(file)
	err = decoder.Decode(config)
	if err != nil {
		logger.Fatal("Failed to decode config file:", err, lager.Data{"config": *config})
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", config.Port))
	if err != nil {
		logger.Fatal("Error listening on port.", err, lager.Data{"port": config.Port})
	}
	defer listener.Close()

	err = ioutil.WriteFile(config.Pidfile, []byte(strconv.Itoa(os.Getpid())), 0644)
	if err != nil {
		logger.Fatal("Cannot write pid to file", err, lager.Data{"pidfile": config.Pidfile})
	}
	logger.Info(fmt.Sprintf("Wrote pidfile to %s", config.Pidfile))

	for _, backend := range config.Backends {
		backendIPs = append(backendIPs, backend.BackendIP)
		backendPorts = append(backendPorts, backend.BackendPort)
		healthcheckPorts = append(healthcheckPorts, backend.HealthcheckPort)
	}

	logger.Info(fmt.Sprintf("Proxy started on port %d\n", config.Port))

	fmt.Printf("Proxy started with configuration: %+v\n", config)

	backends := switchboard.NewBackends(
		backendIPs,
		backendPorts,
		healthcheckPorts,
		logger,
	)

	cluster := switchboard.NewCluster(
		backends,
		time.Millisecond*time.Duration(config.HealthcheckTimeoutInMS),
		logger,
	)

	switchboard := switchboard.New(listener, cluster, logger)

	switchboard.Run()
}
