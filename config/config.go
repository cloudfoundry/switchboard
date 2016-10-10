package config

import (
	"errors"
	"flag"
	"fmt"
	"time"

	"code.cloudfoundry.org/cflager"
	"code.cloudfoundry.org/lager"

	"gopkg.in/validator.v2"

	"github.com/pivotal-cf-experimental/service-config"
)

type Config struct {
	Proxy             Proxy     `yaml:"Proxy" validate:"nonzero"`
	API               API       `yaml:"API" validate:"nonzero"`
	Profiling         Profiling `yaml:"Profiling"`
	StaticDir         string    `yaml:"StaticDir" validate:"nonzero"`
	PidFile           string    `yaml:"PidFile" validate:"nonzero"`
	HealthPort        uint      `yaml:"HealthPort" validate:"nonzero"`
	ConsulCluster     string    `yaml:"ConsulCluster"`
	ConsulServiceName string    `yaml:"ConsulServiceName"`
	Logger            lager.Logger
}

type Profiling struct {
	Enabled bool `yaml:"Enabled"`
	Port    uint `yaml:"Port"`
}

type Proxy struct {
	Port                     uint      `yaml:"Port" validate:"nonzero"`
	Backends                 []Backend `yaml:"Backends" validate:"min=1"`
	HealthcheckTimeoutMillis uint      `yaml:"HealthcheckTimeoutMillis" validate:"nonzero"`
}

type API struct {
	Port       uint   `yaml:"Port" validate:"nonzero"`
	Username   string `yaml:"Username" validate:"nonzero"`
	Password   string `yaml:"Password" validate:"nonzero"`
	ForceHttps bool   `yaml:"ForceHttps"`
}

type Backend struct {
	Host           string `yaml:"Host" validate:"nonzero"`
	Port           uint   `yaml:"Port" validate:"nonzero"`
	StatusPort     uint   `yaml:"StatusPort" validate:"nonzero"`
	StatusEndpoint string `yaml:"StatusEndpoint" validate:"nonzero"`
	Name           string `yaml:"Name" validate:"nonzero"`
}

func (p Proxy) HealthcheckTimeout() time.Duration {
	return time.Duration(p.HealthcheckTimeoutMillis) * time.Millisecond
}

func NewConfig(osArgs []string) (*Config, error) {
	var rootConfig Config

	binaryName := osArgs[0]
	configurationOptions := osArgs[1:]

	serviceConfig := service_config.New()
	flags := flag.NewFlagSet(binaryName, flag.ExitOnError)

	cflager.AddFlags(flags)

	serviceConfig.AddFlags(flags)
	flags.Parse(configurationOptions)

	err := serviceConfig.Read(&rootConfig)

	rootConfig.Logger, _ = cflager.New(binaryName)

	return &rootConfig, err
}

func (c Config) Validate() error {
	rootConfigErr := validator.Validate(c)
	var errString string
	if rootConfigErr != nil {
		errString = formatErrorString(rootConfigErr, "")
	}

	// validator.Validate does not work on nested arrays
	for i, backend := range c.Proxy.Backends {
		backendsErr := validator.Validate(backend)
		if backendsErr != nil {
			errString += formatErrorString(
				backendsErr,
				fmt.Sprintf("Proxy.Backends[%d].", i),
			)
		}
	}

	if len(errString) > 0 {
		return errors.New(fmt.Sprintf("Validation errors: %s\n", errString))
	}
	return nil
}

func formatErrorString(err error, keyPrefix string) string {
	errs := err.(validator.ErrorMap)
	var errsString string
	for fieldName, validationMessage := range errs {
		errsString += fmt.Sprintf("%s%s : %s\n", keyPrefix, fieldName, validationMessage)
	}
	return errsString
}
