package switchboard_test

import (
	"fmt"
	"net"
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
		proxyPort := 10000 + GinkgoParallelNode()
		logger := lager.NewLogger("ProxyRunner test")
		proxyRunner := switchboard.NewProxyRunner(cluster, uint(proxyPort), logger)
		proxyProcess := ifrit.Invoke(proxyRunner)
		proxyProcess.Signal(os.Kill)
		Eventually(proxyProcess.Wait()).Should(Receive())

		_, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", proxyPort))
		Expect(err).To(HaveOccurred())
	})
})
