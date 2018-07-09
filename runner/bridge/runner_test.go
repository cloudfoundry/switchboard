package bridge_test

import (
	"fmt"
	"net"
	"os"
	"time"

	"code.cloudfoundry.org/lager/lagertest"
	"github.com/cloudfoundry-incubator/switchboard/runner/bridge"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
)

var _ = Describe("Bridge Runner", func() {
	It("shuts down gracefully when signalled", func() {
		timeout := 100 * time.Millisecond

		proxyPort := 10000 + GinkgoParallelNode()
		logger := lagertest.NewTestLogger("ProxyRunner test")

		proxyRunner := bridge.NewRunner( uint(proxyPort), timeout, logger)
		proxyProcess := ifrit.Invoke(proxyRunner)

		Eventually(func() error {
			_, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", proxyPort))
			return err
		}).ShouldNot(HaveOccurred())

		proxyProcess.Signal(os.Kill)

		smallEpsilon := 10 * time.Millisecond

		Consistently(func() error {
			_, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", proxyPort))
			return err
		},
			timeout-smallEpsilon,
		).Should(Succeed())

		Eventually(proxyProcess.Wait()).Should(Receive())

		_, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", proxyPort))
		Expect(err).To(HaveOccurred())
	})
})
