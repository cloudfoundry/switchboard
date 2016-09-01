package api_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/switchboard/api"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("ClusterAPI", func() {
	var (
		logger             lager.Logger
		cluster            *api.ClusterAPI
		trafficEnabledChan chan bool
	)

	BeforeEach(func() {
		trafficEnabledChan = make(chan bool, 10)
	})

	JustBeforeEach(func() {
		logger = lagertest.NewTestLogger("Cluster test")
		cluster = api.NewClusterAPI(trafficEnabledChan, logger)
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
