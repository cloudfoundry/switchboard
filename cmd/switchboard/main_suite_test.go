package main_test

import (
	"os"
	"testing"

	"github.com/fraenkel/candiedyaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf-experimental/switchboard/config"
)

func TestSwitchboard(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Switchboard Executable Suite")
}

var switchboardBinPath string
var dummyBackendBinPath string
var dummyHealthcheckBinPath string
var switchboardPort uint
var backendPort uint
var backendPort2 uint
var dummyHealthcheckPort uint
var dummyHealthcheckPort2 uint
var proxyConfigFile string
var proxyConfig config.Proxy

var _ = BeforeSuite(func() {
	var err error
	switchboardBinPath, err = gexec.Build("github.com/pivotal-cf-experimental/switchboard/cmd/switchboard", "-race")
	Ω(err).ShouldNot(HaveOccurred())

	dummyBackendBinPath, err = gexec.Build("github.com/pivotal-cf-experimental/switchboard/cmd/switchboard/internal/dummy_backend", "-race")
	Ω(err).ShouldNot(HaveOccurred())

	dummyHealthcheckBinPath, err = gexec.Build("github.com/pivotal-cf-experimental/switchboard/cmd/switchboard/internal/dummy_healthcheck", "-race")
	Ω(err).ShouldNot(HaveOccurred())

	switchboardPort = uint(39900 + GinkgoParallelNode())
	healthcheckTimeoutInMS := uint(500)

	backendPort = uint(45000 + GinkgoParallelNode())
	backendPort2 = uint(46000 + GinkgoParallelNode())
	dummyHealthcheckPort = uint(45500 + GinkgoParallelNode())
	dummyHealthcheckPort2 = uint(46500 + GinkgoParallelNode())

	backend1 := config.Backend{
		BackendIP:       "localhost",
		BackendPort:     backendPort,
		HealthcheckPort: dummyHealthcheckPort,
	}

	backend2 := config.Backend{
		BackendIP:       "localhost",
		BackendPort:     backendPort2,
		HealthcheckPort: dummyHealthcheckPort2,
	}

	backends := []config.Backend{backend1, backend2}

	proxyConfig = config.Proxy{
		Pidfile:                "/tmp/switchboard.pid",
		Backends:               backends,
		HealthcheckTimeoutInMS: healthcheckTimeoutInMS,
		Port: switchboardPort,
	}

	proxyConfigFile = "/tmp/proxyConfig.yml"

	fileToWrite, err := os.Create(proxyConfigFile)
	if err != nil {
		println("Failed to open file for writing:", err.Error())
		os.Exit(1)
	}

	encoder := candiedyaml.NewEncoder(fileToWrite)
	err = encoder.Encode(proxyConfig)

	if err != nil {
		println("Failed to encode document:", err.Error())
		os.Exit(1)
	}

})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
