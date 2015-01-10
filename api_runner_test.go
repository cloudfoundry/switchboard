package switchboard_test

import (
	"net"
	"os"

	"github.com/pivotal-cf-experimental/switchboard"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
)

var _ = Describe("APIRunner", func() {
	It("shuts down gracefully when signalled", func() {
		apiRunner := switchboard.NewAPIRunner(12345)
		apiProcess := ifrit.Invoke(apiRunner)
		apiProcess.Signal(os.Kill)
		Eventually(apiProcess.Wait()).Should(Receive())

		_, err := net.Dial("tcp", "127.0.0.1:12345")
		Expect(err).To(HaveOccurred())
	})
})
