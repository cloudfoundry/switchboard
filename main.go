package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/consuladapter"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/locket"

	"github.com/cloudfoundry-incubator/switchboard/api"
	"github.com/cloudfoundry-incubator/switchboard/config"
	"github.com/cloudfoundry-incubator/switchboard/domain"
	apirunner "github.com/cloudfoundry-incubator/switchboard/runner/api"
	"github.com/cloudfoundry-incubator/switchboard/runner/bridge"
	"github.com/cloudfoundry-incubator/switchboard/runner/health"
	"github.com/cloudfoundry-incubator/switchboard/runner/monitor"
	consulapi "github.com/hashicorp/consul/api"
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
	arpEntryRemover := monitor.NewARPFlusher(new(monitor.ExecCmdRunner))

	bridgeActiveBackendChan := make(chan *domain.Backend)
	clusterAPIActiveBackendChan := make(chan *domain.Backend)
	activeBackendSubscribers := []chan<- *domain.Backend{
		bridgeActiveBackendChan,
		clusterAPIActiveBackendChan,
	}

	cluster := monitor.NewCluster(
		backends,
		rootConfig.Proxy.HealthcheckTimeout(),
		logger,
		arpEntryRemover,
		activeBackendSubscribers,
	)

	trafficEnabledChan := make(chan bool)

	clusterAPI := api.NewClusterAPI(trafficEnabledChan, clusterAPIActiveBackendChan, logger)
	go clusterAPI.ListenForActiveBackend()

	handler := api.NewHandler(clusterAPI, backends, logger, rootConfig.API, rootConfig.StaticDir)

	members := grouper.Members{
		{
			Name:   "bridge",
			Runner: bridge.NewRunner(bridgeActiveBackendChan, trafficEnabledChan, rootConfig.Proxy.Port, logger),
		},
		{
			Name:   "api",
			Runner: apirunner.NewRunner(rootConfig.API.Port, handler),
		},
		{
			Name:   "monitor",
			Runner: monitor.NewRunner(cluster, logger),
		},
	}

	if rootConfig.HealthPort != rootConfig.API.Port {
		members = append(members, grouper.Member{
			Name:   "health",
			Runner: health.NewRunner(rootConfig.HealthPort),
		})
	}

	if rootConfig.ConsulCluster != "" {
		writePid(logger, rootConfig.PidFile)

		if rootConfig.ConsulServiceName == "" {
			rootConfig.ConsulServiceName = "mysql"
		}

		clock := clock.NewClock()
		consulClient, err := consuladapter.NewClientFromUrl(rootConfig.ConsulCluster)
		if err != nil {
			logger.Fatal("new-consul-client-failed", err)
		}

		lock := locket.NewLock(
			logger,
			consulClient,
			locket.LockSchemaPath(rootConfig.ConsulServiceName+"_lock"),
			[]byte{},
			clock,
			locket.RetryInterval,
			locket.LockTTL,
		)

		registrationRunner := locket.NewRegistrationRunner(logger,
			&consulapi.AgentServiceRegistration{
				Name:  rootConfig.ConsulServiceName,
				Port:  int(rootConfig.Proxy.Port),
				Check: &consulapi.AgentServiceCheck{TTL: "3s"},
			},
			consulClient, locket.RetryInterval, clock)

		members = append([]grouper.Member{{"lock", lock}}, members...)
		members = append(members, grouper.Member{"registration", registrationRunner})
	}

	group := grouper.NewOrdered(os.Interrupt, members)
	process := ifrit.Invoke(sigmon.New(group))

	logger.Info("Proxy started", lager.Data{"proxyConfig": rootConfig.Proxy})

	if rootConfig.ConsulCluster == "" {
		writePid(logger, rootConfig.PidFile)
	}

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
