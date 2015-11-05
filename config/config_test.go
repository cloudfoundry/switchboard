package config_test

import (
	"fmt"

	. "github.com/cloudfoundry-incubator/switchboard/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf-experimental/service-config/test_helpers"
)

var _ = Describe("Config", func() {
	Describe("Validate", func() {

		var (
			rootConfig *Config
			rawConfig  string
		)

		JustBeforeEach(func() {
			osArgs := []string{
				"switchboard",
				fmt.Sprintf("-config=%s", rawConfig),
			}

			var err error
			rootConfig, err = NewConfig(osArgs)
			Expect(err).ToNot(HaveOccurred())
		})

		BeforeEach(func() {
			rawConfig = `{
				"API": {
					"Port": 80,
					"Username": "fake-username",
					"Password": "fake-password",
					"ForceHttps": true
				},
				"Proxy": {
					"Port": 3306,
					"HealthcheckTimeoutMillis": 5000,
					"Backends": [
						{
							"Host": "10.10.10.10",
							"Port": 3306,
							"HealthcheckPort": 9200,
							"Name": "backend-0"
						}
					]
				},
				"ProfilerPort": 6060,
				"HealthPort": 9200,
				"StaticDir": "fake-path",
				"PidFile": "fake-pid-path"
			}`
		})

		It("does not return error on valid config", func() {
			err := rootConfig.Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns an error if API.Port is blank", func() {
			err := test_helpers.IsRequiredField(rootConfig, "API.Port")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if API.Username is blank", func() {
			err := test_helpers.IsRequiredField(rootConfig, "API.Username")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if API.Password is blank", func() {
			err := test_helpers.IsRequiredField(rootConfig, "API.Password")
			Expect(err).ToNot(HaveOccurred())
		})

		It("does not return an error if API.ForceHttps is blank", func() {
			err := test_helpers.IsOptionalField(rootConfig, "API.ForceHttps")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if Proxy.Port is blank", func() {
			err := test_helpers.IsRequiredField(rootConfig, "Proxy.Port")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if Proxy.HealthcheckTimeoutMillis is blank", func() {
			err := test_helpers.IsRequiredField(rootConfig, "Proxy.HealthcheckTimeoutMillis")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if Proxy.Backends is blank", func() {
			err := test_helpers.IsRequiredField(rootConfig, "Proxy.Backends")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if Proxy.Backends.Host is blank", func() {
			err := test_helpers.IsRequiredField(rootConfig, "Proxy.Backends.Host")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if Proxy.Backends.Port is blank", func() {
			err := test_helpers.IsRequiredField(rootConfig, "Proxy.Backends.Port")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if Proxy.Backends.HealthcheckPort is blank", func() {
			err := test_helpers.IsRequiredField(rootConfig, "Proxy.Backends.HealthcheckPort")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if Proxy.Backends.Name is blank", func() {
			err := test_helpers.IsRequiredField(rootConfig, "Proxy.Backends.Name")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if ProfilerPort is blank", func() {
			err := test_helpers.IsRequiredField(rootConfig, "ProfilerPort")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if HealthPort is blank", func() {
			err := test_helpers.IsRequiredField(rootConfig, "HealthPort")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if StaticDir is blank", func() {
			err := test_helpers.IsRequiredField(rootConfig, "StaticDir")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if PidFile is blank", func() {
			err := test_helpers.IsRequiredField(rootConfig, "PidFile")
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
