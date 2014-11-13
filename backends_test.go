package switchboard_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/pivotal-cf-experimental/switchboard"
)

var _ = Describe("Backends", func() {
	var backends switchboard.Backends
	// var backend1 switchboard.Backend
	// var backend2 switchboard.Backend
	// var backend3 switchboard.Backend

	BeforeEach(func() {
		backend_ips := []string{"localhost", "localhost", "localhost"}
		backend_ports := []uint{50000, 50001, 50002}
		healthcheck_ports := []uint{60000, 60001, 60002}
		backends = switchboard.NewBackends(backend_ips, backend_ports, healthcheck_ports, nil)
	})

	Describe("Concurrent operations", func() {
		It("do not result in a race", func() {
			readySetGo := make(chan interface{})

			doneChans := []chan interface{}{
				make(chan interface{}),
				make(chan interface{}),
				make(chan interface{}),
			}

			go func() {
				<-readySetGo
				backends.All()
				close(doneChans[0])
			}()

			go func() {
				<-readySetGo
				backends.Active()
				close(doneChans[1])
			}()

			go func() {
				<-readySetGo
				backends.SetActive(nil)
				close(doneChans[2])
			}()

			close(readySetGo)

			for _, done := range doneChans {
				<-done
			}
		})
	})

	// Describe("Remove", func() {
	//   It("removes only the given bridge", func() {
	//     err := bridges.Remove(bridge2)
	//     Expect(err).NotTo(HaveOccurred())

	//     Expect(bridges.Contains(bridge1)).To(BeTrue())
	//     Expect(bridges.Contains(bridge2)).To(BeFalse())
	//     Expect(bridges.Contains(bridge3)).To(BeTrue())

	//     Expect(bridges.Size()).To(Equal(2))
	//   })

	//   Context("when the bridge cannot be found", func() {
	//     It("returns an error", func() {
	//       err := bridges.Remove(switchboard.NewBridge(&fakes.FakeReadWriteCloser{}, &fakes.FakeReadWriteCloser{}, lager.NewLogger("test")))
	//       Expect(err).To(HaveOccurred())
	//       Expect(err).To(MatchError("Bridge not found"))
	//     })
	//   })
	// })

	// Describe("RemoveAndCloseAll", func() {
	//   BeforeEach(func() {
	//     switchboard.BridgeProvider = func(_, _ io.ReadWriteCloser, _ lager.Logger) switchboard.Bridge {
	//       return &fakes.FakeBridge{}
	//     }
	//   })

	//   AfterEach(func() {
	//     switchboard.BridgeProvider = switchboard.NewBridge
	//   })

	//   It("closes all bridges", func() {
	//     bridges.RemoveAndCloseAll()

	//     Expect(bridge1.(*fakes.FakeBridge).CloseCallCount()).To(Equal(1))
	//     Expect(bridge2.(*fakes.FakeBridge).CloseCallCount()).To(Equal(1))
	//     Expect(bridge3.(*fakes.FakeBridge).CloseCallCount()).To(Equal(1))
	//   })

	//   It("removes all bridges", func() {
	//     bridges.RemoveAndCloseAll()

	//     Expect(bridges.Size()).To(Equal(0))
	//   })
	// })
})
