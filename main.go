package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"

	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/cloudfoundry-incubator/locket"
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

	"time"

	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
)

func main() {

	rootConfig, err := config.NewConfig(os.Args)

	logger := rootConfig.Logger

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

	if _, err := os.Stat(rootConfig.StaticDir); os.IsNotExist(err) {
		logger.Fatal(fmt.Sprintf("staticDir: %s does not exist", rootConfig.StaticDir), nil)
	}

	backends := domain.NewBackends(rootConfig.Proxy.Backends, logger)
	arpManager := monitor.NewArmManager(logger)
	cluster := monitor.NewCluster(
		backends,
		rootConfig.Proxy.HealthcheckTimeout(),
		logger,
		arpManager,
	)

	trafficEnabledChan := make(chan bool)
	bridgeTrafficEnabledChan := make(chan bool)

	domain.BroadcastBool(trafficEnabledChan, []chan<- bool {
		bridgeTrafficEnabledChan,
	})

	clusterApi := api.NewClusterAPI(backends, trafficEnabledChan, logger)

	clusterRouter := bridge.NewClusterRouter(backends)

	handler := api.NewHandler(clusterApi, backends, logger, rootConfig.API, rootConfig.StaticDir)

	members := grouper.Members{
		{
			Name:   "bridge",
			Runner: bridge.NewRunner(clusterRouter, backends, bridgeTrafficEnabledChan, rootConfig.Proxy.Port, logger),
		},
		{
			Name:   "api",
			Runner: apirunner.NewRunner(rootConfig.API.Port, handler, logger),
		},
		{
			Name:   "monitor",
			Runner: monitor.NewRunner(cluster, logger),
		},
	}

	if rootConfig.HealthPort != rootConfig.API.Port {
		members = append(members, grouper.Member{
			Name:   "health",
			Runner: health.NewRunner(rootConfig.HealthPort, logger),
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

	group := grouper.NewOrdered(os.Kill, members)

	process := ifrit.Invoke(group)

	err = waitUntilReady(process, logger)
	if err != nil {
		logger.Fatal("Error starting switchboard", err, lager.Data{"proxyConfig": rootConfig.Proxy})
	}

	logger.Info("Proxy started", lager.Data{"proxyConfig": rootConfig.Proxy})

	if rootConfig.ConsulCluster == "" {
		writePid(logger, rootConfig.PidFile)
	}

	err = <-process.Wait()
	if err != nil {
		logger.Fatal("Switchboard exited unexpectedly", err, lager.Data{"proxyConfig": rootConfig.Proxy})
	}
}

func waitUntilReady(process ifrit.Process, logger lager.Logger) error {
	//we could not find a reliable way for ifrit to report that all processes
	//were ready without error, so we opted to simply report as ready if no errors
	//were thrown within a timeout
	ready := time.After(5 * time.Second)
	select {
	case <-ready:
		logger.Info("All child processes are ready")
		return nil
	case err := <-process.Wait():
		if err == nil {
			//sometimes process will exit early, but will return a nil error
			err = errors.New("Child process exited before becoming ready")
		}
		return err
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
