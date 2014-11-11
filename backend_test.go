package switchboard_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf-experimental/switchboard"
	"github.com/pivotal-golang/lager"
)

type FakeConnection struct{}

func (fc *FakeConnection) Read(buf []byte) (n int, err error) {
	return 0, nil
}

func (fc *FakeConnection) Write(buf []byte) (n int, err error) {
	return 0, nil
}

func (fc *FakeConnection) Close() (err error) {
	return nil
}

type FakeBridge struct {
	wasClosed bool
}

func (fb *FakeBridge) Connect() {}
func (fb *FakeBridge) Close() {
	fb.wasClosed = true
}

var _ = Describe("Backend", func() {
	var backend Backend

	BeforeEach(func() {
		backend = NewBackend("node 0", "10.244.1.2", 3306, 9200)
	})

	Describe("RemoveBridge", func() {
		var bridge1 *ConnectionBridge
		var bridge2 *ConnectionBridge
		var bridge3 *ConnectionBridge

		BeforeEach(func() {
			bridge1 = NewConnectionBridge(&FakeConnection{}, &FakeConnection{}, lager.NewLogger("test"))
			bridge2 = NewConnectionBridge(&FakeConnection{}, &FakeConnection{}, lager.NewLogger("test"))
			bridge3 = NewConnectionBridge(&FakeConnection{}, &FakeConnection{}, lager.NewLogger("test"))
			backend.AddBridge(bridge1)
			backend.AddBridge(bridge2)
			backend.AddBridge(bridge3)
		})

		It("removes only the given bridge", func() {
			err := backend.RemoveBridge(bridge2)
			Expect(err).NotTo(HaveOccurred())

			index, err := backend.IndexOfBridge(bridge2)
			Expect(err).To(HaveOccurred())

			index, err = backend.IndexOfBridge(bridge1)
			Expect(err).NotTo(HaveOccurred())
			Expect(index).To(Equal(0))

			index, err = backend.IndexOfBridge(bridge3)
			Expect(err).NotTo(HaveOccurred())
			Expect(index).To(Equal(1))

			Expect(len(backend.Bridges())).To(Equal(2))
		})

		Context("when the bridge cannot be found", func() {
			It("returns an error", func() {
				err := backend.RemoveBridge(NewConnectionBridge(&FakeConnection{}, &FakeConnection{}, lager.NewLogger("test")))
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("Bridge not found in backend"))
			})
		})
	})

	Describe("RemoveAndCloseAllBridges", func() {
		var bridge1 *FakeBridge
		var bridge2 *FakeBridge
		var bridge3 *FakeBridge

		BeforeEach(func() {
			bridge1 = &FakeBridge{}
			bridge2 = &FakeBridge{}
			bridge3 = &FakeBridge{}
			backend.AddBridge(bridge1)
			backend.AddBridge(bridge2)
			backend.AddBridge(bridge3)
		})

		It("closes all bridges", func() {
			backend.RemoveAndCloseAllBridges()

			Expect(bridge1.wasClosed).To(BeTrue())
			Expect(bridge2.wasClosed).To(BeTrue())
			Expect(bridge3.wasClosed).To(BeTrue())
		})

		It("removes all bridges", func() {
			backend.RemoveAndCloseAllBridges()

			Expect(len(backend.Bridges())).To(Equal(0))
		})
	})

	Describe("IndexOfBridge", func() {
		var bridge1 *FakeBridge
		var bridge2 *FakeBridge
		var bridge3 *FakeBridge
		BeforeEach(func() {
			bridge1 = &FakeBridge{}
			bridge2 = &FakeBridge{}
			bridge3 = &FakeBridge{}
			backend.AddBridge(bridge1)
			backend.AddBridge(bridge2)
			backend.AddBridge(bridge3)
		})

		It("returns the index of the requested bridge", func() {
			index, err := backend.IndexOfBridge(bridge2)
			Expect(index).To(Equal(1))
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns -1 and an error when the bridge is not present", func() {
			index, err := backend.IndexOfBridge(NewConnectionBridge(&FakeConnection{}, &FakeConnection{}, lager.NewLogger("test")))
			Expect(index).To(Equal(-1))
			Expect(err).To(HaveOccurred())
		})
	})
})
