package switchboard_test

import (
	"os"

	"github.com/pivotal-cf-experimental/switchboard"
	"github.com/pivotal-cf-experimental/switchboard/fakes"
	"github.com/pivotal-golang/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
)

var _ = Describe("ProxyRunner", func() {
	It("shuts down gracefully when signalled", func() {
		cluster := &fakes.FakeCluster{}
		proxyRunner := switchboard.NewProxyRunner(cluster, 1234, lager.NewLogger("ProxyRunner test"))
		proxyProcess := ifrit.Invoke(proxyRunner)
		proxyProcess.Signal(os.Kill)
		Eventually(proxyProcess.Wait()).Should(Receive())
	})
})
