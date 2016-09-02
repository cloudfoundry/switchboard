package bridge_test

import (
	"fmt"
	"net"
	"os"

	"github.com/cloudfoundry-incubator/switchboard/runner/bridge"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/ifrit"
)

var _ = Describe("Bridge Runner", func() {
	It("shuts down gracefully when signalled", func() {
		proxyPort := 10000 + GinkgoParallelNode()
		logger := lagertest.NewTestLogger("ProxyRunner test")

		proxyRunner := bridge.NewRunner(nil, nil, uint(proxyPort), logger)
		proxyProcess := ifrit.Invoke(proxyRunner)
		proxyProcess.Signal(os.Kill)
		Eventually(proxyProcess.Wait()).Should(Receive())

		_, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", proxyPort))
		Expect(err).To(HaveOccurred())
	})
})
