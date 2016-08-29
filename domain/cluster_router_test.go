package domain_test

import (
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/switchboard/domain"
	"github.com/cloudfoundry-incubator/switchboard/domain/domainfakes"
)

var _ = Describe("ClusterRouter", func() {
	var (
		backends      *domainfakes.FakeBackends
		clusterRouter *domain.ClusterRouter
		backend       *domainfakes.FakeBackend
	)

	BeforeEach(func() {
		backends = new(domainfakes.FakeBackends)

		backend = new(domainfakes.FakeBackend)
	})

	JustBeforeEach(func() {
		clusterRouter = domain.NewClusterRouter(backends)
	})

	Describe("RouteToBackend", func() {
		var clientConn net.Conn

		BeforeEach(func() {
			clientConn = new(domainfakes.FakeConn)
		})

		It("bridges the client connection to the active backend", func() {
			activeBackend := new(domainfakes.FakeBackend)
			backends.ActiveReturns(activeBackend)

			err := clusterRouter.RouteToBackend(clientConn)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(activeBackend.BridgeCallCount()).To(Equal(1))
			Expect(activeBackend.BridgeArgsForCall(0)).To(Equal(clientConn))
		})

		It("returns an error if there is no active backend", func() {
			backends.ActiveReturns(nil)

			err := clusterRouter.RouteToBackend(clientConn)

			Expect(err).Should(HaveOccurred())
		})
	})
})
