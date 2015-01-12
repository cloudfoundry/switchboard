package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/cloudfoundry-incubator/cf-lager"
	"github.com/pivotal-cf-experimental/switchboard"
	"github.com/pivotal-cf-experimental/switchboard/config"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"

	"github.com/pivotal-golang/lager"
)

var (
	configFlag = flag.String("config", "", "Path to config file")
	pidFile    = flag.String("pidFile", "", "Path to pid file")

	logger lager.Logger
)

func main() {
	flag.Parse()

	logger = cf_lager.New("main")

	proxyConfig, err := config.Load(*configFlag)
	if err != nil {
		logger.Fatal("Error loading config file:", err, lager.Data{"config": *configFlag})
	}

	err = ioutil.WriteFile(*pidFile, []byte(strconv.Itoa(os.Getpid())), 0644)
	if err == nil {
		logger.Info(fmt.Sprintf("Wrote pidFile to %s", *pidFile))
	} else {
		logger.Fatal("Cannot write pid to file", err, lager.Data{"pidFile": *pidFile})
	}

	backends := switchboard.NewBackends(proxyConfig.Backends, logger)
	cluster := switchboard.NewCluster(
		backends,
		proxyConfig.HealthcheckTimeout(),
		logger,
	)

	proxyRunner := switchboard.NewProxyRunner(cluster, proxyConfig.Port, logger)
	apiRunner := switchboard.NewAPIRunner(proxyConfig.APIPort, backends)
	group := grouper.NewParallel(os.Kill, grouper.Members{
		grouper.Member{"proxy", proxyRunner},
		grouper.Member{"api", apiRunner},
	})
	process := ifrit.Invoke(group)

	logger.Info(fmt.Sprintf("Proxy started on port %d\n", proxyConfig.Port))
	logger.Info(fmt.Sprintf("Proxy started with configuration: %+v\n", proxyConfig))

	err = <-process.Wait()
	if err != nil {
		logger.Fatal("Error starting switchboard", err, lager.Data{"proxyConfig": proxyConfig})
	}
}
