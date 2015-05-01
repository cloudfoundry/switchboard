package main_test

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/cloudfoundry-incubator/switchboard/config"
	"github.com/cloudfoundry-incubator/candiedyaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const (
	switchboardPackage = "github.com/cloudfoundry-incubator/switchboard/"
)

func TestSwitchboard(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Switchboard Executable Suite")
}

var switchboardBinPath string
var switchboardPort uint
var switchboardAPIPort uint
var switchboardProfilerPort uint
var switchboardHealthPort uint
var backends []config.Backend
var configPath string
var rootConfig config.Config
var proxyConfig config.Proxy
var apiConfig config.API
var pidFile string
var staticDir string

var _ = BeforeSuite(func() {
	var err error
	switchboardBinPath, err = gexec.Build(switchboardPackage, "-race")
	Expect(err).NotTo(HaveOccurred())

	tempDir, err := ioutil.TempDir(os.TempDir(), "switchboard")
	Expect(err).NotTo(HaveOccurred())

	pidFile = filepath.Join(tempDir, "switchboard.pid")

	configPath = filepath.Join(tempDir, "proxyConfig.yml")
})

func initConfig() {
	switchboardPort = uint(39900 + GinkgoParallelNode())
	switchboardAPIPort = uint(39000 + GinkgoParallelNode())
	switchboardProfilerPort = uint(6060 + GinkgoParallelNode())
	switchboardHealthPort = uint(6160 + GinkgoParallelNode())

	backend1 := config.Backend{
		Host:            "localhost",
		Port:            uint(45000 + GinkgoParallelNode()),
		HealthcheckPort: uint(45500 + GinkgoParallelNode()),
		Name:            "backend-0",
	}

	backend2 := config.Backend{
		Host:            "localhost",
		Port:            uint(46000 + GinkgoParallelNode()),
		HealthcheckPort: uint(46500 + GinkgoParallelNode()),
		Name:            "backend-1",
	}

	backends = []config.Backend{backend1, backend2}

	proxyConfig = config.Proxy{
		Backends:                 backends,
		HealthcheckTimeoutMillis: 500,
		Port: switchboardPort,
	}
	apiConfig = config.API{
		Port:     switchboardAPIPort,
		Username: "username",
		Password: "password",
	}
	rootConfig = config.Config{
		Proxy:        proxyConfig,
		API:          apiConfig,
		ProfilerPort: switchboardProfilerPort,
		HealthPort:   switchboardHealthPort,
	}
}

func writeConfig() {
	fileToWrite, err := os.Create(configPath)
	Ω(err).ShouldNot(HaveOccurred())

	encoder := candiedyaml.NewEncoder(fileToWrite)
	err = encoder.Encode(rootConfig)
	Ω(err).ShouldNot(HaveOccurred())

	testDir := getDirOfCurrentFile()
	staticDir = filepath.Join(testDir, "static")
}

func getDirOfCurrentFile() string {
	_, filename, _, _ := runtime.Caller(1)
	return path.Dir(filename)
}

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
