package switchboard_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf-experimental/switchboard"
	"github.com/pivotal-cf-experimental/switchboard/fakes"
	"github.com/pivotal-golang/lager"
)

var _ = Describe("Bridges", func() {
	var bridges Bridges

	BeforeEach(func() {
		bridges = NewBridges()
	})

	Describe("Remove", func() {
		var bridge1 *ConnectionBridge
		var bridge2 *ConnectionBridge
		var bridge3 *ConnectionBridge

		BeforeEach(func() {
			bridge1 = NewConnectionBridge(&fakes.FakeReadWriteCloser{}, &fakes.FakeReadWriteCloser{}, lager.NewLogger("test"))
			bridge2 = NewConnectionBridge(&fakes.FakeReadWriteCloser{}, &fakes.FakeReadWriteCloser{}, lager.NewLogger("test"))
			bridge3 = NewConnectionBridge(&fakes.FakeReadWriteCloser{}, &fakes.FakeReadWriteCloser{}, lager.NewLogger("test"))
			bridges.Add(bridge1)
			bridges.Add(bridge2)
			bridges.Add(bridge3)
		})

		It("removes only the given bridge", func() {
			err := bridges.Remove(bridge2)
			Expect(err).NotTo(HaveOccurred())

			Expect(bridges.Contains(bridge1)).To(BeTrue())
			Expect(bridges.Contains(bridge2)).To(BeFalse())
			Expect(bridges.Contains(bridge3)).To(BeTrue())

			Expect(bridges.Size()).To(Equal(2))
		})

		Context("when the bridge cannot be found", func() {
			It("returns an error", func() {
				err := bridges.Remove(NewConnectionBridge(&fakes.FakeReadWriteCloser{}, &fakes.FakeReadWriteCloser{}, lager.NewLogger("test")))
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("Bridge not found"))
			})
		})
	})

	Describe("RemoveAndCloseAll", func() {
		var bridge1 *fakes.FakeBridge
		var bridge2 *fakes.FakeBridge
		var bridge3 *fakes.FakeBridge

		BeforeEach(func() {
			bridge1 = &fakes.FakeBridge{}
			bridge2 = &fakes.FakeBridge{}
			bridge3 = &fakes.FakeBridge{}
			bridges.Add(bridge1)
			bridges.Add(bridge2)
			bridges.Add(bridge3)
		})

		It("closes all bridges", func() {
			bridges.RemoveAndCloseAll()

			Expect(bridge1.CloseCallCount()).To(Equal(1))
			Expect(bridge2.CloseCallCount()).To(Equal(1))
			Expect(bridge3.CloseCallCount()).To(Equal(1))
		})

		It("removes all bridges", func() {
			bridges.RemoveAndCloseAll()

			Expect(bridges.Size()).To(Equal(0))
		})
	})
})
