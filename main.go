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
	"github.com/cloudfoundry-incubator/switchboard/health"
	"github.com/cloudfoundry-incubator/switchboard/proxy"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"

	"github.com/pivotal-cf-experimental/service-config"
	"github.com/pivotal-golang/lager"
)

func main() {
	serviceConfig := service_config.New()

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	pidFile := flags.String("pidFile", "", "Path to pid file")
	staticDir := flags.String("staticDir", "", "Path to directory containing static UI")
	serviceConfig.AddFlags(flags)
	cf_lager.AddFlags(flags)
	flags.Parse(os.Args[1:])

	logger, _ := cf_lager.New("Switchboard")

	var rootConfig config.Config
	err := serviceConfig.Read(&rootConfig)
	if err != nil {
		logger.Fatal("Error reading config:", err)
	}

	err = rootConfig.Validate()
	if err != nil {
		logger.Fatal("Error validating config:", err, lager.Data{"config": rootConfig})
	}

	go func() {
		logger.Info(fmt.Sprintf("Starting profiling server on port %d", rootConfig.ProfilerPort))
		err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", rootConfig.ProfilerPort), nil)
		if err != nil {
			logger.Error("profiler failed with error", err)
		}
	}()

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

	members := grouper.Members{
		{
			Name:   "proxy",
			Runner: proxy.NewRunner(cluster, rootConfig.Proxy.Port, logger),
		},
		{
			Name:   "api",
			Runner: api.NewRunner(rootConfig.API.Port, handler, logger),
		},
	}

	if rootConfig.HealthPort != rootConfig.API.Port {
		members = append(members, grouper.Member{
			Name:   "health",
			Runner: health.NewRunner(rootConfig.HealthPort, logger),
		})
	}

	group := grouper.NewParallel(os.Kill, members)
	process := ifrit.Invoke(group)

	logger.Info("Proxy started", lager.Data{"proxyConfig": rootConfig.Proxy})

	err = <-process.Wait()
	if err != nil {
		logger.Fatal("Error starting switchboard", err, lager.Data{"proxyConfig": rootConfig.Proxy})
	}
}
