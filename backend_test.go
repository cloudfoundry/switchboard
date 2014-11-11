package switchboard_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf-experimental/switchboard"
)

var _ = Describe("Backend", func() {
	var backend Backend

	BeforeEach(func() {
		backend = NewBackend("1.2.3.4", 3306, 9902, nil)
	})

	Describe("HealthcheckUrl", func() {
		It("has the correct scheme, backend ip and health check port", func() {
			healthcheckURL := backend.HealthcheckUrl()
			Expect(healthcheckURL).To(Equal("http://1.2.3.4:9902"))
		})
	})
})
