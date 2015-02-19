package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"

	"github.com/cloudfoundry-incubator/cf-lager"
	"github.com/cloudfoundry-incubator/switchboard/api"
	"github.com/cloudfoundry-incubator/switchboard/config"
	"github.com/cloudfoundry-incubator/switchboard/domain"
	"github.com/cloudfoundry-incubator/switchboard/proxy"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"

	"github.com/pivotal-golang/lager"
)

func main() {
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	configFile := flags.String("configFile", "", "Path to config file")
	pidFile := flags.String("pidFile", "", "Path to pid file")
	staticDir := flags.String("staticDir", "", "Path to directory containing static UI")
	cf_lager.AddFlags(flags)
	flags.Parse(os.Args[1:])

	logger, _ := cf_lager.New("Switchboard")

	go func() {
		logger.Info("Starting pprof server")
		http.ListenAndServe("localhost:6060", nil)
	}()

	rootConfig, err := config.Load(*configFile)
	if err != nil {
		logger.Fatal("Error loading config file:", err, lager.Data{"config": *configFile})
	}

	err = ioutil.WriteFile(*pidFile, []byte(strconv.Itoa(os.Getpid())), 0644)
	if err == nil {
		logger.Info(fmt.Sprintf("Wrote pidFile to %s", *pidFile))
	} else {
		logger.Fatal("Cannot write pid to file", err, lager.Data{"pidFile": *pidFile})
	}

	if *staticDir == "" {
		logger.Fatal("staticDir flag not provided", nil)
	}

	if _, err := os.Stat(*staticDir); os.IsNotExist(err) {
		logger.Fatal(fmt.Sprintf("staticDir: %s does not exist", *staticDir), nil)
	}

	backends := domain.NewBackends(rootConfig.Proxy.Backends, logger)
	cluster := domain.NewCluster(
		backends,
		rootConfig.Proxy.HealthcheckTimeout(),
		logger,
	)

	handler := api.NewHandler(backends, logger, rootConfig.API, *staticDir)

	group := grouper.NewParallel(os.Kill, grouper.Members{
		grouper.Member{"proxy", proxy.NewRunner(cluster, rootConfig.Proxy.Port, logger)},
		grouper.Member{"api", api.NewRunner(rootConfig.API.Port, handler, logger)},
	})
	process := ifrit.Invoke(group)

	logger.Info("Proxy started", lager.Data{"proxyConfig": rootConfig.Proxy})

	err = <-process.Wait()
	if err != nil {
		logger.Fatal("Error starting switchboard", err, lager.Data{"proxyConfig": rootConfig.Proxy})
	}
}
