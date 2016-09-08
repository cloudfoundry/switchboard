package api_test

import (
	"github.com/cloudfoundry-incubator/switchboard/api"
	"github.com/cloudfoundry-incubator/switchboard/api/apifakes"
	"github.com/cloudfoundry-incubator/switchboard/domain"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("BackendsIndex", func() {
	var (
		logger lager.Logger

		fakeClusterManager *apifakes.FakeClusterManager
		clusterAsJSON      *api.ClusterJSON

		backends api.Backends

		backend0 *domain.Backend
		backend1 *domain.Backend
	)

	BeforeEach(func() {
		fakeClusterManager = &apifakes.FakeClusterManager{}

		logger = lagertest.NewTestLogger("BackendsIndex test")

		backend0 = domain.NewBackend(
			"backend-0",
			"backend-0-host",
			23000,
			23001,
			"backend-0-endpoint",
			logger,
		)

		backend1 = domain.NewBackend(
			"backend-1",
			"backend-1-host",
			23010,
			23011,
			"backend-1-endpoint",
			logger,
		)

		backends = api.Backends{
			backend0,
			backend1,
		}

		backend1JSON := api.BackendJSON{
			Host: "backend-1-host",
			Port: 23010,
			Name: "backend-1",
		}

		clusterAsJSON = &api.ClusterJSON{
			ActiveBackend:  &backend1JSON,
			TrafficEnabled: true,
		}
	})

	JustBeforeEach(func() {
		fakeClusterManager.AsJSONReturns(*clusterAsJSON)
	})

	Describe("AsV0JSON", func() {
		It("returns the backends", func() {
			v0backendResponses := backends.AsV0JSON(fakeClusterManager)
			Expect(v0backendResponses).To(HaveLen(2))

			Expect(v0backendResponses[0].TrafficEnabled).To(BeTrue())
			Expect(v0backendResponses[1].TrafficEnabled).To(BeTrue())

			Expect(v0backendResponses[0].Name).To(Equal(backend0.AsJSON().Name))
			Expect(v0backendResponses[1].Name).To(Equal(backend1.AsJSON().Name))

			Expect(v0backendResponses[0].Host).To(Equal(backend0.AsJSON().Host))
			Expect(v0backendResponses[1].Host).To(Equal(backend1.AsJSON().Host))

			Expect(v0backendResponses[0].Healthy).To(Equal(backend0.AsJSON().Healthy))
			Expect(v0backendResponses[1].Healthy).To(Equal(backend1.AsJSON().Healthy))
		})

		It("returns the active backend from the cluster manager", func() {
			v0backendResponses := backends.AsV0JSON(fakeClusterManager)
			Expect(v0backendResponses).To(HaveLen(2))

			Expect(v0backendResponses[0].Active).To(BeFalse())
			Expect(v0backendResponses[1].Active).To(BeTrue())
		})
	})
})
