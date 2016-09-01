package api_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/switchboard/api"
	"github.com/cloudfoundry-incubator/switchboard/api/apifakes"
	"github.com/cloudfoundry-incubator/switchboard/domain"
	"github.com/cloudfoundry-incubator/switchboard/domain/domainfakes"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("ClusterAPI", func() {
	var (
		backends                     *apifakes.FakeBackends
		backendSlice                 []*domainfakes.FakeBackend
		logger                       lager.Logger
		cluster                      *api.ClusterAPI
		backend1, backend2, backend3 *domainfakes.FakeBackend
		trafficEnabledChan           chan bool
	)

	BeforeEach(func() {
		trafficEnabledChan = make(chan bool, 10)
		backends = new(apifakes.FakeBackends)

		backend1 = new(domainfakes.FakeBackend)
		backend1.AsJSONReturns(domain.BackendJSON{Host: "10.10.1.2"})
		backend1.HealthcheckUrlReturns("backend1")

		backend2 = new(domainfakes.FakeBackend)
		backend2.AsJSONReturns(domain.BackendJSON{Host: "10.10.2.2"})
		backend2.HealthcheckUrlReturns("backend2")

		backend3 = new(domainfakes.FakeBackend)
		backend3.AsJSONReturns(domain.BackendJSON{Host: "10.10.3.2"})
		backend3.HealthcheckUrlReturns("backend3")

		backendSlice = []*domainfakes.FakeBackend{backend1, backend2, backend3}

		backends.AllStub = func() <-chan domain.Backend {
			c := make(chan domain.Backend, 3)

			for _, b := range backendSlice {
				c <- b
			}
			close(c)

			return c
		}
	})

	JustBeforeEach(func() {
		logger = lagertest.NewTestLogger("Cluster test")
		cluster = api.NewClusterAPI(backends, trafficEnabledChan, logger)
	})

	Describe("EnableTraffic", func() {
		var (
			message string
		)

		BeforeEach(func() {
			message = "some message"
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

		It("records that traffic is enabled", func() {
			cluster.EnableTraffic(message)

			clusterJSON := cluster.AsJSON()

			Expect(clusterJSON.TrafficEnabled).To(BeTrue())
		})

		It("publishes that traffic is enabled", func() {
			cluster.EnableTraffic(message)

			Eventually(trafficEnabledChan).Should(Receive(BeTrue()))
		})
	})

	Describe("DisableTraffic", func() {
		var (
			message string
		)

		BeforeEach(func() {
			message = "some message"
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
