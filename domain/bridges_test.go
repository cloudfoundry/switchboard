package domain_test

import (
	"net"

	"github.com/cloudfoundry-incubator/switchboard/domain"
	"github.com/cloudfoundry-incubator/switchboard/models/modelsfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/cloudfoundry-incubator/switchboard/models"
	"github.com/cloudfoundry-incubator/switchboard/domain/domainfakes"
)

var _ = Describe("Bridges", func() {
	var (
		bridges        models.Bridges
		bridge1        models.Bridge
		bridge2        models.Bridge
		bridge3        models.Bridge
		bridgeProvider func(net.Conn, net.Conn, lager.Logger) models.Bridge
	)

	BeforeEach(func() {
		logger := lagertest.NewTestLogger("Bridges Test")
		bridges = domain.NewBridges(logger)
		bridgeProvider = domain.BridgeProvider
	})

	AfterEach(func() {
		domain.BridgeProvider = bridgeProvider
	})

	JustBeforeEach(func() {
		bridge1 = bridges.Create(nil, nil)
		bridge2 = bridges.Create(nil, nil)
		bridge3 = bridges.Create(nil, nil)
	})

	Describe("Concurrent operations", func() {
		It("do not result in a race", func() {
			readySetGo := make(chan interface{})

			doneChans := []chan interface{}{
				make(chan interface{}),
				make(chan interface{}),
				make(chan interface{}),
				make(chan interface{}),
				make(chan interface{}),
			}

			go func() {
				<-readySetGo
				bridges.Create(nil, nil)
				close(doneChans[0])
			}()

			go func() {
				<-readySetGo
				bridges.Contains(bridge1)
				close(doneChans[1])
			}()

			go func() {
				<-readySetGo
				bridges.Remove(bridge2)
				close(doneChans[2])
			}()

			go func() {
				<-readySetGo
				bridges.Size()
				close(doneChans[3])
			}()

			go func() {
				<-readySetGo
				bridges.RemoveAndCloseAll()
				close(doneChans[4])
			}()

			close(readySetGo)

			for _, done := range doneChans {
				<-done
			}
		})
	})

	Describe("Remove", func() {
		It("removes only the given bridge", func() {
			err := bridges.Remove(bridge2)
			Expect(err).NotTo(HaveOccurred())

			Expect(bridges.Contains(bridge1)).To(BeTrue())
			Expect(bridges.Contains(bridge2)).To(BeFalse())
			Expect(bridges.Contains(bridge3)).To(BeTrue())

			Expect(bridges.Size()).To(BeNumerically("==", 2))
		})

		Context("when the bridge cannot be found", func() {
			It("returns an error", func() {
				err := bridges.Remove(domain.NewBridge(new(domainfakes.FakeConn), new(domainfakes.FakeConn), nil))
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("Bridge not found"))
			})
		})
	})

	Describe("RemoveAndCloseAll", func() {
		BeforeEach(func() {
			domain.BridgeProvider = func(client, backend net.Conn, logger lager.Logger) models.Bridge {
				return new(modelsfakes.FakeBridge)
			}
		})

		It("closes all bridges", func() {
			bridges.RemoveAndCloseAll()

			Expect(bridge1.(*modelsfakes.FakeBridge).CloseCallCount()).To(Equal(1))
			Expect(bridge2.(*modelsfakes.FakeBridge).CloseCallCount()).To(Equal(1))
			Expect(bridge3.(*modelsfakes.FakeBridge).CloseCallCount()).To(Equal(1))
		})

		It("removes all bridges", func() {
			bridges.RemoveAndCloseAll()

			Expect(bridges.Size()).To(BeNumerically("==", 0))
		})
	})
})
