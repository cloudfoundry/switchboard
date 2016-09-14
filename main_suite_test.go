package main_test

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/cloudfoundry-incubator/switchboard/config"
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

var (
	switchboardBinPath      string
	proxyPort               uint
	switchboardAPIPort      uint
	switchboardProfilerPort uint
	switchboardHealthPort   uint
	backends                []config.Backend
	configPath              string
	rootConfig              config.Config
	proxyConfig             config.Proxy
	apiConfig               config.API
	pidFile                 string
	tempDir                 string
	staticDir               string
)

var _ = BeforeSuite(func() {
	var err error
	switchboardBinPath, err = gexec.Build(switchboardPackage, "-race")
	Expect(err).NotTo(HaveOccurred())

	tempDir, err := ioutil.TempDir(os.TempDir(), "switchboard")
	Expect(err).NotTo(HaveOccurred())

	configPath = filepath.Join(tempDir, "proxyConfig.yml")

	testDir := getDirOfCurrentFile()
	staticDir = filepath.Join(testDir, "static")
})

func initConfig() {
	pidFileFile, _ := ioutil.TempFile(tempDir, "switchboard.pid")
	pidFileFile.Close()
	pidFile = pidFileFile.Name()
	os.Remove(pidFile)

	proxyPort = uint(39900 + GinkgoParallelNode())
	switchboardAPIPort = uint(39000 + GinkgoParallelNode())
	switchboardProfilerPort = uint(6060 + GinkgoParallelNode())
	switchboardHealthPort = uint(6160 + GinkgoParallelNode())

	backend1 := config.Backend{
		Host:           "localhost",
		Port:           uint(45000 + GinkgoParallelNode()),
		StatusPort:     uint(45500 + GinkgoParallelNode()),
		StatusEndpoint: "galera_healthcheck",
		Name:           "backend-0",
	}

	backend2 := config.Backend{
		Host:           "localhost",
		Port:           uint(46000 + GinkgoParallelNode()),
		StatusPort:     uint(46500 + GinkgoParallelNode()),
		StatusEndpoint: "galera_healthcheck",
		Name:           "backend-1",
	}

	backends = []config.Backend{backend1, backend2}

	proxyConfig = config.Proxy{
		Backends:                 backends,
		HealthcheckTimeoutMillis: 500,
		Port: proxyPort,
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
		PidFile:      pidFile,
		StaticDir:    staticDir,
	}
}

func writeConfig() {
	fileToWrite, err := os.Create(configPath)
	Expect(err).NotTo(HaveOccurred())

	b, err := yaml.Marshal(rootConfig)
	Expect(err).NotTo(HaveOccurred())

	_, err = fileToWrite.Write(b)
	Expect(err).NotTo(HaveOccurred())
}

func getDirOfCurrentFile() string {
	_, filename, _, _ := runtime.Caller(1)
	return path.Dir(filename)
}

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
