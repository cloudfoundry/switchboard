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

	Describe("RemoveBridge", func() {
		var bridge1 *ConnectionBridge
		var bridge2 *ConnectionBridge
		var bridge3 *ConnectionBridge

		BeforeEach(func() {
			bridge1 = NewConnectionBridge(&fakes.FakeReadWriteCloser{}, &fakes.FakeReadWriteCloser{}, lager.NewLogger("test"))
			bridge2 = NewConnectionBridge(&fakes.FakeReadWriteCloser{}, &fakes.FakeReadWriteCloser{}, lager.NewLogger("test"))
			bridge3 = NewConnectionBridge(&fakes.FakeReadWriteCloser{}, &fakes.FakeReadWriteCloser{}, lager.NewLogger("test"))
			bridges.AddBridge(bridge1)
			bridges.AddBridge(bridge2)
			bridges.AddBridge(bridge3)
		})

		It("removes only the given bridge", func() {
			err := bridges.RemoveBridge(bridge2)
			Expect(err).NotTo(HaveOccurred())

			index, err := bridges.IndexOfBridge(bridge2)
			Expect(err).To(HaveOccurred())

			index, err = bridges.IndexOfBridge(bridge1)
			Expect(err).NotTo(HaveOccurred())
			Expect(index).To(Equal(0))

			index, err = bridges.IndexOfBridge(bridge3)
			Expect(err).NotTo(HaveOccurred())
			Expect(index).To(Equal(1))

			Expect(len(bridges.Bridges())).To(Equal(2))
		})

		Context("when the bridge cannot be found", func() {
			It("returns an error", func() {
				err := bridges.RemoveBridge(NewConnectionBridge(&fakes.FakeReadWriteCloser{}, &fakes.FakeReadWriteCloser{}, lager.NewLogger("test")))
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("Bridge not found"))
			})
		})
	})

	Describe("RemoveAndCloseAllBridges", func() {
		var bridge1 *fakes.FakeBridge
		var bridge2 *fakes.FakeBridge
		var bridge3 *fakes.FakeBridge

		BeforeEach(func() {
			bridge1 = &fakes.FakeBridge{}
			bridge2 = &fakes.FakeBridge{}
			bridge3 = &fakes.FakeBridge{}
			bridges.AddBridge(bridge1)
			bridges.AddBridge(bridge2)
			bridges.AddBridge(bridge3)
		})

		It("closes all bridges", func() {
			bridges.RemoveAndCloseAllBridges()

			Expect(bridge1.CloseCallCount()).To(Equal(1))
			Expect(bridge2.CloseCallCount()).To(Equal(1))
			Expect(bridge3.CloseCallCount()).To(Equal(1))
		})

		It("removes all bridges", func() {
			bridges.RemoveAndCloseAllBridges()

			Expect(len(bridges.Bridges())).To(Equal(0))
		})
	})

	Describe("IndexOfBridge", func() {
		var bridge1 *fakes.FakeBridge
		var bridge2 *fakes.FakeBridge
		var bridge3 *fakes.FakeBridge
		BeforeEach(func() {
			bridge1 = &fakes.FakeBridge{}
			bridge2 = &fakes.FakeBridge{}
			bridge3 = &fakes.FakeBridge{}
			bridges.AddBridge(bridge1)
			bridges.AddBridge(bridge2)
			bridges.AddBridge(bridge3)
		})

		It("returns the index of the requested bridge", func() {
			index, err := bridges.IndexOfBridge(bridge2)
			Expect(index).To(Equal(1))
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns -1 and an error when the bridge is not present", func() {
			index, err := bridges.IndexOfBridge(NewConnectionBridge(&fakes.FakeReadWriteCloser{}, &fakes.FakeReadWriteCloser{}, lager.NewLogger("test")))
			Expect(index).To(Equal(-1))
			Expect(err).To(HaveOccurred())
		})
	})
})
