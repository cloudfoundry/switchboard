package config_test

import (
	"io/ioutil"

	"github.com/pivotal-cf-experimental/switchboard/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Describe("Load", func() {
		It("returns an error when an empty file path is provided", func() {
			_, err := config.Load("")
			Expect(err).To(HaveOccurred())
		})

		It("returns an error when path is provided to a file that doesn't exist", func() {
			_, err := config.Load("/file/does/not/exist.yml")
			Expect(err).To(HaveOccurred())
		})

		It("returns an error when a field of incorrect type cannot be decoded", func() {
			invalidConfig, err := ioutil.TempFile("", "invalidConfig.yml")
			Expect(err).NotTo(HaveOccurred())

			_, err = invalidConfig.WriteString(`"Proxy": {"HealthcheckTimeoutMillis": "NotAnInteger"}`)
			Expect(err).NotTo(HaveOccurred())

			err = invalidConfig.Close()
			Expect(err).NotTo(HaveOccurred())

			_, err = config.Load(invalidConfig.Name())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("NotAnInteger"))
		})

		It("returns an error if the config file does not contain a proxy", func() {
			_, err := config.Load("testConfigs/emptyProxy.yml")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Proxy"))
		})

		It("returns an error if the config file does not contain an api", func() {
			_, err := config.Load("testConfigs/emptyAPI.yml")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("API"))
		})

		It("returns an error if one of the proxy fields is empty", func() {
			_, err := config.Load("testConfigs/invalidProxyFields.yml")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Proxy.Port"))
			Expect(err.Error()).To(ContainSubstring("Proxy.Backends"))
			Expect(err.Error()).To(ContainSubstring("Proxy.HealthcheckTimeoutMillis"))
		})

		It("returns an error if one of the Backends fields is empty", func() {
			_, err := config.Load("testConfigs/invalidBackends.yml")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Proxy.Backends[0].Host"))
			Expect(err.Error()).To(ContainSubstring("Proxy.Backends[0].Port"))
			Expect(err.Error()).To(ContainSubstring("Proxy.Backends[0].HealthcheckPort"))
			Expect(err.Error()).To(ContainSubstring("Proxy.Backends[0].Name"))
		})

		It("returns an error if one of the API fields is empty", func() {
			_, err := config.Load("testConfigs/invalidAPIFields.yml")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("API.Port"))
			Expect(err.Error()).To(ContainSubstring("API.Username"))
			Expect(err.Error()).To(ContainSubstring("API.Password"))
		})
	})
})
