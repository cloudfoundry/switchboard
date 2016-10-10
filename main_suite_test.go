package main_test

import (
	"path"
	"runtime"
	"testing"

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
	switchboardBinPath string
)

var _ = BeforeSuite(func() {
	var err error
	switchboardBinPath, err = gexec.Build(switchboardPackage, "-race")
	Expect(err).NotTo(HaveOccurred())
})

func getDirOfCurrentFile() string {
	_, filename, _, _ := runtime.Caller(1)
	return path.Dir(filename)
}

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
