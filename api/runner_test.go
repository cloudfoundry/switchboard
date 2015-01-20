package api_test

import (
	"fmt"
	"net"
	"os"

	"github.com/pivotal-cf-experimental/switchboard/api"
	"github.com/pivotal-cf-experimental/switchboard/domain/fakes"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
)

var _ = Describe("APIRunner", func() {
	It("shuts down gracefully when signalled", func() {
		apiPort := 10000 + GinkgoParallelNode()
		backends := &fakes.FakeBackends{}
		logger := lagertest.NewTestLogger("APIRunner Test")
		apiRunner := api.NewRunner(uint(apiPort), api.NewHandler(backends), logger)
		apiProcess := ifrit.Invoke(apiRunner)
		apiProcess.Signal(os.Kill)
		Eventually(apiProcess.Wait()).Should(Receive())

		_, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", apiPort))
		Expect(err).To(HaveOccurred())
	})
})
