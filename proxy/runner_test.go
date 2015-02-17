package proxy_test

import (
	"fmt"
	"net"
	"os"

	"github.com/cloudfoundry-incubator/switchboard/domain/fakes"
	"github.com/cloudfoundry-incubator/switchboard/proxy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/ifrit"
)

var _ = Describe("ProxyRunner", func() {
	It("shuts down gracefully when signalled", func() {
		cluster := &fakes.FakeCluster{}
		proxyPort := 10000 + GinkgoParallelNode()
		logger := lagertest.NewTestLogger("ProxyRunner test")
		proxyRunner := proxy.NewRunner(cluster, uint(proxyPort), logger)
		proxyProcess := ifrit.Invoke(proxyRunner)
		proxyProcess.Signal(os.Kill)
		Eventually(proxyProcess.Wait()).Should(Receive())

		_, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", proxyPort))
		Expect(err).To(HaveOccurred())
	})
})
