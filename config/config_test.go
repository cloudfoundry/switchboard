package config_test

import (
	. "github.com/cloudfoundry-incubator/switchboard/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Describe("Validate", func() {
		It("returns an error if the config file does not contain a proxy", func() {
            config := Config{
                API: API{
                    Port: 0,
                    Username: "",
                    Password: "",
                },
                ProfilerPort: 0,
                HealthPort: 0,
            }

			err := config.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Proxy"))
		})

		It("returns an error if the config file does not contain an api", func() {
            config := Config{
                Proxy: Proxy{
                    Port: 0,
                    HealthcheckTimeoutMillis: 0,
                },
                ProfilerPort: 0,
                HealthPort: 0,
            }

            err := config.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("API"))
		})

		It("returns an error if one of the proxy fields is empty", func() {
            config := Config{
                Proxy: Proxy{
                    Port: 0,
                    HealthcheckTimeoutMillis: 0,
                    Backends: []Backend{
                        {
                            Host: "",
                            Port: 0,
                            HealthcheckPort: 0,
                            Name: "",
                        },
                    },
                },
                API: API{
                    Port: 0,
                    Username: "",
                    Password: "",
                },
                ProfilerPort: 0,
                HealthPort: 0,
            }

            err := config.Validate()
			Expect(err.Error()).To(ContainSubstring("Proxy.Port"))
			Expect(err.Error()).To(ContainSubstring("Proxy.Backends"))
			Expect(err.Error()).To(ContainSubstring("Proxy.HealthcheckTimeoutMillis"))
		})

		It("returns an error if one of the Backends fields is empty", func() {
            config := Config{
                Proxy: Proxy{
                    Port: 0,
                    HealthcheckTimeoutMillis: 0,
                    Backends: []Backend{
                        {
                            Host: "",
                            Port: 0,
                            HealthcheckPort: 0,
                            Name: "",
                        },
                    },
                },
                API: API{
                    Port: 0,
                    Username: "",
                    Password: "",
                },
                ProfilerPort: 0,
                HealthPort: 0,
            }

            err := config.Validate()
			Expect(err.Error()).To(ContainSubstring("Proxy.Backends[0].Host"))
			Expect(err.Error()).To(ContainSubstring("Proxy.Backends[0].Port"))
			Expect(err.Error()).To(ContainSubstring("Proxy.Backends[0].HealthcheckPort"))
			Expect(err.Error()).To(ContainSubstring("Proxy.Backends[0].Name"))
		})

		It("returns an error if one of the API fields is empty", func() {
            config := Config{
                Proxy: Proxy{
                    Port: 0,
                    HealthcheckTimeoutMillis: 0,
                    Backends: []Backend{
                        {
                            Host: "",
                            Port: 0,
                            HealthcheckPort: 0,
                            Name: "",
                        },
                    },
                },
                API: API{
                    Port: 0,
                    Username: "",
                    Password: "",
                },
                ProfilerPort: 0,
                HealthPort: 0,
            }

            err := config.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("API.Port"))
			Expect(err.Error()).To(ContainSubstring("API.Username"))
			Expect(err.Error()).To(ContainSubstring("API.Password"))
		})

		It("returns an error if the config file does not contain a profile port", func() {
            config := Config{
                Proxy: Proxy{
                    Port: 0,
                    HealthcheckTimeoutMillis: 0,
                },
                API: API{
                    Port: 0,
                    Username: "",
                    Password: "",
                },
                HealthPort: 0,
            }

            err := config.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("ProfilerPort"))
		})

		It("returns an error if the config file does not contain a health port", func() {
            config := Config{
                Proxy: Proxy{
                    Port: 0,
                    HealthcheckTimeoutMillis: 0,
                },
                API: API{
                    Port: 0,
                    Username: "",
                    Password: "",
                },
                ProfilerPort: 0,
            }

            err := config.Validate()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("HealthPort"))
		})
	})
})
