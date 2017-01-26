package config_test

import (
	"fmt"
	"time"

	. "github.com/cloudfoundry-incubator/switchboard/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf-experimental/service-config/test_helpers"
)

var _ = Describe("Config", func() {
	Describe("Proxy methods", func() {
		Describe("HealthcheckTimeout", func() {
			It("returns timeout in millis", func() {
				Expect(Proxy{HealthcheckTimeoutMillis: 10}.HealthcheckTimeout()).To(Equal(10 * time.Millisecond))
			})
		})

		Describe("ShutdownDelay", func() {
			It("returns delay in seconds", func() {
				Expect(Proxy{ShutdownDelaySeconds: 10}.ShutdownDelay()).To(Equal(10 * time.Second))
			})
		})
	})

	Describe("Validate", func() {
		var (
			rootConfig    *Config
			rawConfig     string
			rawConfigFile string
		)

		JustBeforeEach(func() {
			osArgs := []string{
				"switchboard",
				fmt.Sprintf("-configPath=%s", rawConfigFile),
			}

			var err error
			rootConfig, err = NewConfig(osArgs)
			Expect(err).ToNot(HaveOccurred())
		})

		BeforeEach(func() {
			rawConfig = `{
				"API": {
					"Port": "80",
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
							"StatusPort": 9200,
							"StatusEndpoint": "galera_healthcheck",
							"Name": "backend-0"
						}
					]
				},
				"Profiling": {
					"Enabled": true,
					"Port": 6060
				},
				"HealthPort": 9200,
				"StaticDir": "fake-path",
				"PidFile": "fake-pid-path"
			}`
			rawConfigFile = "fixtures/validConfig.yml"
		})

		It("does not return error on valid config", func() {
			err := rootConfig.Validate()
			Expect(err).ToNot(HaveOccurred())
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

		It("does not return an error if ConsulCluster is blank", func() {
			err := test_helpers.IsOptionalField(rootConfig, "ConsulCluster")
			Expect(err).ToNot(HaveOccurred())
		})

		It("does not return an error if ConsulServiceName is blank", func() {
			err := test_helpers.IsOptionalField(rootConfig, "ConsulServiceName")
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

		It("returns an error if Proxy.Backends.StatusPort is blank", func() {
			err := test_helpers.IsRequiredField(rootConfig, "Proxy.Backends.StatusPort")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if Proxy.Backends.StatusEndpoint is blank", func() {
			err := test_helpers.IsRequiredField(rootConfig, "Proxy.Backends.StatusEndpoint")
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns an error if Proxy.Backends.Name is blank", func() {
			err := test_helpers.IsRequiredField(rootConfig, "Proxy.Backends.Name")
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
