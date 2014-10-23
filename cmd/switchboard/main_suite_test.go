package main_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func TestSwitchboard(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Switchboard Main Suite")
}

var switchboardBinPath string
var dummyListenerBinPath string
var dummyHealthCheckBinPath string
var switchboardPort uint
var backendPort uint
var dummyHealthCheckPort uint

var _ = BeforeSuite(func() {
	var err error
	switchboardBinPath, err = gexec.Build("github.com/pivotal-cf-experimental/switchboard/cmd/switchboard", "-race")
	Ω(err).ShouldNot(HaveOccurred())

	dummyListenerBinPath, err = gexec.Build("github.com/pivotal-cf-experimental/switchboard/cmd/dummy_listener", "-race")
	Ω(err).ShouldNot(HaveOccurred())

	dummyHealthCheckBinPath, err = gexec.Build("github.com/pivotal-cf-experimental/switchboard/cmd/dummy_healthcheck", "-race")
	Ω(err).ShouldNot(HaveOccurred())

	switchboardPort = uint(39900 + GinkgoParallelNode())
	backendPort = uint(45000 + GinkgoParallelNode())
	dummyHealthCheckPort = uint(46000 + GinkgoParallelNode())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
