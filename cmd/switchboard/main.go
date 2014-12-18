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
	"github.com/pivotal-cf-experimental/switchboard"
	"github.com/pivotal-cf-experimental/switchboard/config"

	"github.com/pivotal-golang/lager"
)

var (
	configFlag = flag.String("config", "", "Path to config file")

	backendIPs                     []string
	backendPorts, healthcheckPorts []uint
	logger                         lager.Logger
)

func main() {
	flag.Parse()

	logger = cf_lager.New("main")

	proxyConfig, err := config.Load(*configFlag)
	if err != nil {
		logger.Fatal("Error loading config file:", err, lager.Data{"config": *configFlag})
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", proxyConfig.Port))
	if err != nil {
		logger.Fatal("Error listening on port.", err, lager.Data{"port": proxyConfig.Port})
	}
	defer listener.Close()

	err = ioutil.WriteFile(proxyConfig.Pidfile, []byte(strconv.Itoa(os.Getpid())), 0644)
	if err != nil {
		logger.Fatal("Cannot write pid to file", err, lager.Data{"pidfile": proxyConfig.Pidfile})
	}
	logger.Info(fmt.Sprintf("Wrote pidfile to %s", proxyConfig.Pidfile))

	for _, backend := range proxyConfig.Backends {
		backendIPs = append(backendIPs, backend.BackendIP)
		backendPorts = append(backendPorts, backend.BackendPort)
		healthcheckPorts = append(healthcheckPorts, backend.HealthcheckPort)
	}

	logger.Info(fmt.Sprintf("Proxy started on port %d\n", proxyConfig.Port))
	logger.Info(fmt.Sprintf("Proxy started with configuration: %+v\n", proxyConfig))

	backends := switchboard.NewBackends(
		backendIPs,
		backendPorts,
		healthcheckPorts,
		logger,
	)

	cluster := switchboard.NewCluster(
		backends,
		time.Millisecond*time.Duration(proxyConfig.HealthcheckTimeoutInMS),
		logger,
	)

	switchboard := switchboard.New(listener, cluster, logger)

	switchboard.Run()
}
