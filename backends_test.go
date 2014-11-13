package switchboard_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/switchboard"
)

var _ = Describe("Backends", func() {
	var (
		backends          switchboard.Backends
		backend_ips       []string
		backend_ports     []uint
		healthcheck_ports []uint
	)

	var backendChanToSlice = func(c <-chan switchboard.Backend) []switchboard.Backend {
		var result []switchboard.Backend
		for b := range c {
			result = append(result, b)
		}
		return result
	}

	BeforeEach(func() {
		backend_ips = []string{"localhost", "localhost", "localhost"}
		backend_ports = []uint{50000, 50001, 50002}
		healthcheck_ports = []uint{60000, 60001, 60002}
		backends = switchboard.NewBackends(backend_ips, backend_ports, healthcheck_ports, nil)
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

			go func() {
				<-readySetGo
				backends.SetHealthy(nil)
				close(doneChans[3])
			}()

			go func() {
				<-readySetGo
				backends.SetUnhealthy(nil)
				close(doneChans[4])
			}()

			go func() {
				<-readySetGo
				backends.Healthy()
				close(doneChans[5])
			}()

			close(readySetGo)

			for _, done := range doneChans {
				<-done
			}
		})
	})

	Describe("All", func() {
		It("returns a constant list of backends", func() {
			i := 0
			for backend := range backends.All() {
				currentBackend := switchboard.NewBackend(backend_ips[i], backend_ports[i], healthcheck_ports[i], nil)
				i++
				Expect(currentBackend).To(Equal(backend))
			}
		})
	})

	Describe("Active", func() {
		It("returns the currently active backend", func() {
			currentActive := switchboard.NewBackend(backend_ips[0], backend_ports[0], healthcheck_ports[0], nil)
			Expect(currentActive).To(Equal(backends.Active()))
		})
	})

	Describe("SetActive", func() {
		var backend switchboard.Backend
		var active switchboard.Backend

		BeforeEach(func() {
			active = backends.Active()

			for b := range backends.All() {
				if b != active {
					backend = b
					break
				}
			}
		})

		It("sets the active backend", func() {
			Expect(backends.SetActive(backend)).NotTo(HaveOccurred())
			Expect(backends.Active()).To(Equal(backend))

		})
	})

	Describe("SetHealthy", func() {
		var unhealthy switchboard.Backend

		BeforeEach(func() {
			unhealthy = backendChanToSlice(backends.Healthy())[0]
			backends.SetUnhealthy(unhealthy)
		})

		It("sets the backend to be healthy", func() {
			Expect(len(backendChanToSlice(backends.Healthy()))).To(Equal(2))
			backends.SetHealthy(unhealthy)
			Expect(len(backendChanToSlice(backends.Healthy()))).To(Equal(3))
		})
	})

	Describe("SetUnhealthy", func() {
		var healthy switchboard.Backend

		BeforeEach(func() {
			healthy = backendChanToSlice(backends.Healthy())[0]
		})

		It("sets the backend to be healthy", func() {
			Expect(len(backendChanToSlice(backends.Healthy()))).To(Equal(3))
			backends.SetUnhealthy(healthy)
			Expect(len(backendChanToSlice(backends.Healthy()))).To(Equal(2))
		})
	})

	Describe("Healthy", func() {
		It("sets the backend to be healthy", func() {
			healthy := backendChanToSlice(backends.Healthy())
			numHealthy := 3
			Expect(len(healthy)).To(Equal(numHealthy))

			for _, b := range healthy {
				backends.SetUnhealthy(b)
				numHealthy--
				healthy = backendChanToSlice(backends.Healthy())
				Expect(len(healthy)).To(Equal(numHealthy))
			}
		})
	})
})
