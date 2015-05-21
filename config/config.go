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
	Proxy        Proxy `validate:"nonzero"`
	API          API   `validate:"nonzero"`
	ProfilerPort uint  `validate:"nonzero"`
	HealthPort   uint  `validate:"nonzero"`
}

type Proxy struct {
	Port                     uint      `validate:"nonzero"`
	Backends                 []Backend `validate:"min=1"`
	HealthcheckTimeoutMillis uint      `validate:"nonzero"`
}

type API struct {
	Port       uint   `validate:"nonzero"`
	Username   string `validate:"nonzero"`
	Password   string `validate:"nonzero"`
	ForceHttps bool
}

type Backend struct {
	Host            string `validate:"nonzero"`
	Port            uint   `validate:"nonzero"`
	HealthcheckPort uint   `validate:"nonzero"`
	Name            string `validate:"nonzero"`
}

func (p Proxy) HealthcheckTimeout() time.Duration {
	return time.Duration(p.HealthcheckTimeoutMillis) * time.Millisecond
}
