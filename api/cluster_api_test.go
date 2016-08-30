package api_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/switchboard/domain"
	"github.com/cloudfoundry-incubator/switchboard/domain/domainfakes"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/cloudfoundry-incubator/switchboard/api"
)

var _ = Describe("ClusterAPI", func() {
	var (
		backends                     *domainfakes.FakeBackends
		backendSlice                 []*domainfakes.FakeBackend
		logger                       lager.Logger
		cluster                      *api.ClusterAPI
		backend1, backend2, backend3 *domainfakes.FakeBackend
	)

	BeforeEach(func() {
		backends = new(domainfakes.FakeBackends)

		backend1 = new(domainfakes.FakeBackend)
		backend1.AsJSONReturns(domain.BackendJSON{Host: "10.10.1.2"})
		backend1.HealthcheckUrlReturns("backend1")

		backend2 = new(domainfakes.FakeBackend)
		backend2.AsJSONReturns(domain.BackendJSON{Host: "10.10.2.2"})
		backend2.HealthcheckUrlReturns("backend2")
		backend2.TrafficEnabledReturns(true)

		backend3 = new(domainfakes.FakeBackend)
		backend3.AsJSONReturns(domain.BackendJSON{Host: "10.10.3.2"})
		backend3.HealthcheckUrlReturns("backend3")
		backend3.TrafficEnabledReturns(true)

		backendSlice = []*domainfakes.FakeBackend{backend1, backend2, backend3}

		backends.AllStub = func() <-chan domain.Backend {
			c := make(chan domain.Backend)
			go func() {
				c <- backend1
				c <- backend2
				c <- backend3
				close(c)
			}()
			return c
		}

		backends.AnyReturns(backend1)
	})

	JustBeforeEach(func() {
		logger = lagertest.NewTestLogger("Cluster test")
		cluster = api.NewClusterAPI(backends, logger)
	})

	Describe("EnableTraffic", func() {
		var (
			message string
		)

		BeforeEach(func() {
			message = "some message"
		})

		It("calls EnableTraffic on all the backends", func() {
			cluster.EnableTraffic(message)

			for _, backend := range backendSlice {
				Expect(backend.EnableTrafficCallCount()).To(Equal(1))
			}
		})

		It("records the message", func() {
			cluster.EnableTraffic(message)

			clusterJSON := cluster.AsJSON()

			Expect(clusterJSON.Message).To(Equal(message))
		})

		It("records the current time", func() {
			beforeTime := time.Now()
			cluster.EnableTraffic(message)
			afterTime := time.Now()

			clusterJSON := cluster.AsJSON()

			Expect(clusterJSON.LastUpdated.After(beforeTime)).To(BeTrue())
			Expect(clusterJSON.LastUpdated.Before(afterTime)).To(BeTrue())
		})
	})

	Describe("DisableTraffic", func() {
		var (
			message string
		)

		BeforeEach(func() {
			message = "some message"
		})

		It("calls DisableTraffic on all the backends", func() {
			cluster.DisableTraffic(message)

			for _, backend := range backendSlice {
				Expect(backend.DisableTrafficCallCount()).To(Equal(1))
			}
		})

		It("records the message", func() {
			cluster.DisableTraffic(message)

			clusterJSON := cluster.AsJSON()

			Expect(clusterJSON.Message).To(Equal(message))
		})

		It("records the current time", func() {
			beforeTime := time.Now()
			cluster.DisableTraffic(message)
			afterTime := time.Now()

			clusterJSON := cluster.AsJSON()

			Expect(clusterJSON.LastUpdated.After(beforeTime)).To(BeTrue())
			Expect(clusterJSON.LastUpdated.Before(afterTime)).To(BeTrue())
		})
	})
})
