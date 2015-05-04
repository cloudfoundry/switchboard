package health_test

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/cloudfoundry-incubator/switchboard/health"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
)

var _ = Describe("HealthRunner", func() {

	var (
		healthPort     int
		logger         *lagertest.TestLogger
		healthRunner   health.Runner
		healthProcess  ifrit.Process
		startupTimeout = 5 * time.Second
	)

	BeforeEach(func() {

		healthPort = 10000 + GinkgoParallelNode()

		logger = lagertest.NewTestLogger("HealthRunner Test")
		healthRunner = health.NewRunner(uint(healthPort), logger)
		healthProcess = ifrit.Invoke(healthRunner)
		isReady := healthProcess.Ready()
		Eventually(isReady, startupTimeout).Should(BeClosed(), "Error starting Health Runner")
	})

	AfterEach(func() {
		healthProcess.Signal(os.Kill)
		err := <-healthProcess.Wait()
		Expect(err).ToNot(HaveOccurred())
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
