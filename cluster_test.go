package switchboard_test

import (
	"net"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf-experimental/switchboard"
	"github.com/pivotal-cf-experimental/switchboard/fakes"
	"github.com/pivotal-golang/lager"
)

var _ = Describe("Cluster", func() {
	var backends *fakes.FakeBackends
	var logger lager.Logger
	var cluster switchboard.Cluster

	BeforeEach(func() {
		backends = &fakes.FakeBackends{}
		logger = lager.NewLogger("Cluster test")
		cluster = switchboard.NewCluster(backends, time.Second, logger)
	})

	Describe("RouteToBackend", func() {
		var clientConn net.Conn

		BeforeEach(func() {
			clientConn = &fakes.FakeConn{}
		})

		It("bridges the client connection to the active backend", func() {
			activeBackend := &fakes.FakeBackend{}
			backends.ActiveReturns(activeBackend)

			err := cluster.RouteToBackend(clientConn)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(activeBackend.BridgeCallCount()).To(Equal(1))
			Expect(activeBackend.BridgeArgsForCall(0)).To(Equal(clientConn))
		})

		It("returns an error if there is no active backend", func() {
			backends.ActiveReturns(nil)

			err := cluster.RouteToBackend(clientConn)

			Expect(err).Should(HaveOccurred())
		})
	})
})
