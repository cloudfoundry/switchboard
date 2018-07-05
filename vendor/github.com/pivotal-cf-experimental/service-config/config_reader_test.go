package service_config_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/service-config"
)

var _ = Describe("ConfigReader", func() {
	const invalidYAML = `Count: INVALID`
	const simpleYAML = `Name: test-user`
	const nestedYAML = `---
Name: userName
Password: ppp
School: 
  Name: UB
  Location: Buffalo
`
	const partialYAML = `---
Name: yaml-name
`
	const nestedPartialYAML = `---
Name: user-name
Password: password
School:
  Location: nested-partialYAML
`

	type ConfigSimple struct {
		Name string `yaml:"Name"`
	}
	type ConfigInvalid struct {
		Count int `yaml:"Count"`
	}
	type School struct {
		Name     string `yaml:"Name"`
		Location string `yaml:"Location"`
	}

	type ConfigNested struct {
		Name     string `yaml:"Name"`
		Password string `yaml:"Password"`
		School   School `yaml:"School"`
	}

	Describe("Read", func() {

		It("unmarshal a config with one field", func() {
			reader := service_config.NewReader([]byte(simpleYAML))

			var simpleConfig ConfigSimple
			err := reader.Read(&simpleConfig)
			Expect(err).NotTo(HaveOccurred())

			Expect(simpleConfig).To(Equal(ConfigSimple{
				Name: "test-user",
			}))
		})

		It("unmarshal a config with nested fields", func() {
			reader := service_config.NewReader([]byte(nestedYAML))

			var nestedConfig ConfigNested
			err := reader.Read(&nestedConfig)
			Expect(err).NotTo(HaveOccurred())

			Expect(nestedConfig).To(Equal(ConfigNested{
				Name:     "userName",
				Password: "ppp",
				School: School{
					Name:     "UB",
					Location: "Buffalo",
				},
			}))
		})

		It("returns an error for unmarshalling a config without a valid YAML syntax", func() {
			reader := service_config.NewReader([]byte(invalidYAML))

			var invalidConfig ConfigInvalid
			err := reader.Read(&invalidConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unmarshaling config"))
		})
	})

	Describe("ReadWithDefaults", func() {

		Context("with empty defaults", func() {

			var defaultConfig ConfigSimple

			BeforeEach(func() {
				defaultConfig = ConfigSimple{}
			})

			It("returns unmodified config", func() {
				reader := service_config.NewReader([]byte(simpleYAML))

				var simpleConfig ConfigSimple
				err := reader.ReadWithDefaults(&simpleConfig, defaultConfig)
				Expect(err).NotTo(HaveOccurred())

				Expect(simpleConfig).To(Equal(ConfigSimple{
					Name: "test-user",
				}))
			})
		})

		It("adds default top-level field when property is not present", func() {

			defaultConfig := School{
				Name:     "default-name",
				Location: "default-location",
			}
			reader := service_config.NewReader([]byte(partialYAML))

			var schoolConfig School
			err := reader.ReadWithDefaults(&schoolConfig, defaultConfig)
			Expect(err).NotTo(HaveOccurred())

			Expect(schoolConfig).To(Equal(School{
				Name:     "yaml-name",
				Location: "default-location",
			}))
		})

		It("adds default nested field when property is not present", func() {

			defaultConfig := ConfigNested{
				Name:     "default-username",
				Password: "default-password",
				School: School{
					Name:     "default-schoolname",
					Location: "default-schoolLocation",
				},
			}

			reader := service_config.NewReader([]byte(nestedPartialYAML))

			var nestedConfig ConfigNested
			err := reader.ReadWithDefaults(&nestedConfig, defaultConfig)
			Expect(err).NotTo(HaveOccurred())

			Expect(nestedConfig).To(Equal(ConfigNested{
				Name:     "user-name",
				Password: "password",
				School: School{
					Name:     "default-schoolname",
					Location: "nested-partialYAML",
				},
			}))
		})
	})
})
