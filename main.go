package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"

	"code.cloudfoundry.org/lager"

	"github.com/cloudfoundry-incubator/switchboard/api"
	"github.com/cloudfoundry-incubator/switchboard/apiaggregator"
	"github.com/cloudfoundry-incubator/switchboard/config"
	"github.com/cloudfoundry-incubator/switchboard/domain"
	apirunner "github.com/cloudfoundry-incubator/switchboard/runner/api"
	apiaggregatorrunner "github.com/cloudfoundry-incubator/switchboard/runner/apiaggregator"
	"github.com/cloudfoundry-incubator/switchboard/runner/bridge"
	"github.com/cloudfoundry-incubator/switchboard/runner/health"
	"github.com/cloudfoundry-incubator/switchboard/runner/monitor"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
)

func main() {
	rootConfig, err := config.NewConfig(os.Args)

	logger := rootConfig.Logger

	err = rootConfig.Validate()
	if err != nil {
		logger.Fatal("Error validating config:", err, lager.Data{"config": rootConfig})
	}

	go func() {
		if !rootConfig.Profiling.Enabled {
			logger.Info("Profiling disabled")
			return
		}

		logger.Info("Starting profiling server", lager.Data{
			"port": rootConfig.Profiling.Port,
		})
		err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", rootConfig.Profiling.Port), nil)
		if err != nil {
			logger.Error("profiler failed with error", err)
		}
	}()

	if _, err := os.Stat(rootConfig.StaticDir); os.IsNotExist(err) {
		logger.Fatal(fmt.Sprintf("staticDir: %s does not exist", rootConfig.StaticDir), nil)
	}

	backends := domain.NewBackends(rootConfig.Proxy.Backends, logger)

	bridgeActiveBackendChan := make(chan *domain.Backend)
	clusterAPIActiveBackendChan := make(chan *domain.Backend)
	activeBackendSubscribers := []chan<- *domain.Backend{
		bridgeActiveBackendChan,
		clusterAPIActiveBackendChan,
	}

	clusterMonitor := monitor.NewClusterMonitor(
		backends,
		rootConfig.Proxy.HealthcheckTimeout(),
		logger,
		activeBackendSubscribers,
		true,
	)

	trafficEnabledChan := make(chan bool)

	clusterAPI := api.NewClusterAPI(trafficEnabledChan, clusterAPIActiveBackendChan, logger)
	go clusterAPI.ListenForActiveBackend()

	handler := api.NewHandler(clusterAPI, backends, logger, rootConfig.API, rootConfig.StaticDir)
	aggregatorHandler := apiaggregator.NewHandler(logger, rootConfig.API)

	members := grouper.Members{
		{
			Name:   "bridge",
			Runner: bridge.NewRunner(bridgeActiveBackendChan, trafficEnabledChan, rootConfig.Proxy.Port, rootConfig.Proxy.ShutdownDelay(), logger),
		},
		{
			Name:   "api-aggregator",
			Runner: apiaggregatorrunner.NewRunner(rootConfig.API.AggregatorPort, aggregatorHandler),
		},
		{
			Name:   "api",
			Runner: apirunner.NewRunner(rootConfig.API.Port, handler),
		},
		{
			Name:   "monitor",
			Runner: monitor.NewRunner(clusterMonitor, logger),
		},
	}

	if rootConfig.HealthPort != rootConfig.API.Port {
		members = append(members, grouper.Member{
			Name:   "health",
			Runner: health.NewRunner(rootConfig.HealthPort),
		})
	}

	group := grouper.NewOrdered(os.Interrupt, members)
	process := ifrit.Invoke(sigmon.New(group))

	logger.Info("Proxy started", lager.Data{"proxyConfig": rootConfig.Proxy})

	writePid(logger, rootConfig.PidFile)

	err = <-process.Wait()
	if err != nil {
		logger.Fatal("Switchboard exited unexpectedly", err, lager.Data{"proxyConfig": rootConfig.Proxy})
	}
}

func writePid(logger lager.Logger, pidFile string) {
	err := ioutil.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0644)
	if err == nil {
		logger.Info(fmt.Sprintf("Wrote pidFile to %s", pidFile))
	} else {
		logger.Fatal("Cannot write pid to file", err, lager.Data{"pidFile": pidFile})
	}
}
