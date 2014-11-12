package main_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func TestSwitchboard(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Switchboard Executable Suite")
}

var switchboardBinPath string
var dummyListenerBinPath string
var dummyHealthcheckBinPath string
var switchboardPort uint
var backendPort uint
var backendPort2 uint
var dummyHealthcheckPort uint
var dummyHealthcheckPort2 uint
var pidfile string

var _ = BeforeSuite(func() {
	var err error
	switchboardBinPath, err = gexec.Build("github.com/pivotal-cf-experimental/switchboard/cmd/switchboard", "-race")
	Ω(err).ShouldNot(HaveOccurred())

	dummyListenerBinPath, err = gexec.Build("github.com/pivotal-cf-experimental/switchboard/cmd/switchboard/internal/dummy_listener", "-race")
	Ω(err).ShouldNot(HaveOccurred())

	dummyHealthcheckBinPath, err = gexec.Build("github.com/pivotal-cf-experimental/switchboard/cmd/switchboard/internal/dummy_healthcheck", "-race")
	Ω(err).ShouldNot(HaveOccurred())

	switchboardPort = uint(39900 + GinkgoParallelNode())
	backendPort = uint(45000 + GinkgoParallelNode())
	backendPort2 = uint(46000 + GinkgoParallelNode())
	dummyHealthcheckPort = uint(45500 + GinkgoParallelNode())
	dummyHealthcheckPort2 = uint(46500 + GinkgoParallelNode())
	pidfile = "/tmp/switchboard.pid"
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
