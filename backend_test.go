package switchboard_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/switchboard"
	"github.com/pivotal-cf-experimental/switchboard/fakes"
	"github.com/pivotal-golang/lager"
)

var _ = Describe("Backend", func() {
	var backend switchboard.Backend
	var bridges *fakes.FakeBridges

	BeforeEach(func() {
		bridges = &fakes.FakeBridges{}

		switchboard.BridgesProvider = func(logger lager.Logger) switchboard.Bridges {
			return bridges
		}

		backend = switchboard.NewBackend("1.2.3.4", 3306, 9902, nil)
	})

	AfterEach(func() {
		switchboard.BridgesProvider = switchboard.NewBridges
	})

	Describe("HealthcheckUrl", func() {
		It("has the correct scheme, backend ip and health check port", func() {
			healthcheckURL := backend.HealthcheckUrl()
			Expect(healthcheckURL).To(Equal("http://1.2.3.4:9902"))
		})
	})

	Describe("SeverConnections", func() {
		It("removes and closes all bridges", func() {
			backend.SeverConnections()
			Expect(bridges.RemoveAndCloseAllCallCount()).To(Equal(1))
		})
	})
})
