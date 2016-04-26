package config

import (
	"errors"
	"fmt"
	"time"

	"gopkg.in/validator.v2"
)

func (c Config) Validate() error {
	rootConfigErr := validator.Validate(c)
	var errString string
	if rootConfigErr != nil {
		errString = formatErrorString(rootConfigErr, "")
	}

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

type Config struct {
	Proxy        Proxy  `yaml:"Proxy" validate:"nonzero"`
	API          API    `yaml:"API" validate:"nonzero"`
	ProfilerPort uint   `yaml:"ProfilerPort" validate:"nonzero"`
	HealthPort   uint   `yaml:"HealthPort" validate:"nonzero"`
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
	Host            string `yaml:"Host" validate:"nonzero"`
	Port            uint   `yaml:"Port" validate:"nonzero"`
	HealthcheckPort uint   `yaml:"HealthcheckPort" validate:"nonzero"`
	Name            string `yaml:"Name" validate:"nonzero"`
}

func (p Proxy) HealthcheckTimeout() time.Duration {
	return time.Duration(p.HealthcheckTimeoutMillis) * time.Millisecond
}
