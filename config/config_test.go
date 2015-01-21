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

		It("returns an error when the filepath cannot be decoded", func() {
			invalidConfig, err := ioutil.TempFile("", "invalidConfig.yml")
			Expect(err).NotTo(HaveOccurred())

			_, err = invalidConfig.WriteString(`"Proxy": {"HealthcheckTimeoutMillis": "NotAnInteger"}`)
			Expect(err).NotTo(HaveOccurred())

			err = invalidConfig.Close()
			Expect(err).NotTo(HaveOccurred())

			_, err = config.Load(invalidConfig.Name())
			Expect(err).To(HaveOccurred())
		})
	})
})
