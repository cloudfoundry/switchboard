package domain_test

import (
	"net"

	"github.com/cloudfoundry-incubator/switchboard/domain"
	"github.com/cloudfoundry-incubator/switchboard/domain/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("Backend", func() {
	var backend domain.Backend
	var bridges *fakes.FakeBridges

	BeforeEach(func() {
		bridges = &fakes.FakeBridges{}

		domain.BridgesProvider = func(lager.Logger) domain.Bridges {
			return bridges
		}

		logger := lagertest.NewTestLogger("Backend test")
		backend = domain.NewBackend("backend-0", "1.2.3.4", 3306, 9902, logger)
	})

	AfterEach(func() {
		domain.BridgesProvider = domain.NewBridges
	})

	Describe("HealthcheckUrl", func() {
		It("has the correct protocol, backend host and health check port", func() {
			healthcheckURL := backend.HealthcheckUrl()
			Expect(healthcheckURL).To(Equal("http://1.2.3.4:9902"))
		})
	})

	Describe("SeverConnections", func() {
		It("removes and closes all bridges", func() {
			backend.SeverConnections()
			Expect(bridges.RemoveAndCloseAllCallCount()).To(Equal(1))
		})
	})

	Describe("Bridge", func() {
		var backendConn *fakes.FakeConn
		var clientConn *fakes.FakeConn

		var dialErr error
		var dialedProtocol, dialedAddress string
		var bridge *fakes.FakeBridge
		var connectReadyChan, disconnectChan chan interface{}

		BeforeEach(func() {
			bridge = &fakes.FakeBridge{}

			connectReadyChan = make(chan interface{})
			disconnectChan = make(chan interface{})

			bridge.ConnectStub = func(connectReadyChan, disconnectChan chan interface{}) func() {
				return func() {
					close(connectReadyChan)
					<-disconnectChan
				}
			}(connectReadyChan, disconnectChan)

			bridges.CreateReturns(bridge)

			clientConn = &fakes.FakeConn{}
			backendConn = &fakes.FakeConn{}

			clientAddr := &fakes.FakeAddr{}
			backendAddr := &fakes.FakeAddr{}

			clientConn.RemoteAddrReturns(clientAddr)
			backendConn.RemoteAddrReturns(backendAddr)

			dialErr = nil
			dialedAddress = ""

			domain.Dialer = func(protocol, address string) (net.Conn, error) {
				dialedProtocol = protocol
				dialedAddress = address
				return backendConn, dialErr
			}
		})

		AfterEach(func() {
			domain.Dialer = net.Dial
		})

		It("dials the backend address", func(done Done) {
			defer close(done)
			defer close(disconnectChan)

			err := backend.Bridge(clientConn)
			Expect(err).NotTo(HaveOccurred())

			Expect(dialedProtocol).To(Equal("tcp"))
			Expect(dialedAddress).To(Equal("1.2.3.4:3306"))
		})

		It("asynchronously creates and connects to a bridge", func(done Done) {
			defer close(done)
			defer close(disconnectChan)

			err := backend.Bridge(clientConn)
			Expect(err).NotTo(HaveOccurred())

			<-connectReadyChan

			Expect(bridges.CreateCallCount()).Should(Equal(1))
			actualClientConn, actualBackendConn := bridges.CreateArgsForCall(0)
			Expect(actualClientConn).To(Equal(clientConn))
			Expect(actualBackendConn).To(Equal(backendConn))

			Expect(bridge.ConnectCallCount()).To(Equal(1))
		})

		Context("when the bridge is disconnected", func() {
			It("removes the bridge", func(done Done) {
				defer close(done)

				err := backend.Bridge(clientConn)
				Expect(err).NotTo(HaveOccurred())

				<-connectReadyChan

				Consistently(bridges.RemoveCallCount).Should(Equal(0))

				close(disconnectChan)

				Eventually(bridges.RemoveCallCount).Should(Equal(1))
				Expect(bridges.RemoveArgsForCall(0)).To(Equal(bridge))
			}, 2)
		})
	})
})
