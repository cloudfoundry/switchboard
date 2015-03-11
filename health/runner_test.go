package health_test

import (
	"fmt"
	"net"
	"os"

	"github.com/cloudfoundry-incubator/switchboard/health"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
)

var _ = Describe("HealthRunner", func() {

	var (
		healthPort    = 10000 + GinkgoParallelNode()
		logger        *lagertest.TestLogger
		healthRunner  health.Runner
		healthProcess ifrit.Process
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("HealthRunner Test")
		healthRunner = health.NewRunner(uint(healthPort), logger)
		healthProcess = ifrit.Invoke(healthRunner)
	})

	AfterEach(func() {
		healthProcess.Signal(os.Kill)
		<-healthProcess.Wait()
	})

	Context("when the runner is running", func() {
		It("accepts connections on health port", func() {
			conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", healthPort))
			Expect(err).ToNot(HaveOccurred())

			err = conn.Close()
			Expect(err).ToNot(HaveOccurred())
		})
	})

	It("shuts down gracefully when signalled", func() {
		healthProcess.Signal(os.Kill)
		Eventually(healthProcess.Wait()).Should(Receive())

		_, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", healthPort))
		Expect(err).To(HaveOccurred())
	})
})
