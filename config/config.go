package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/fraenkel/candiedyaml"
	"gopkg.in/validator.v2"
)

func Load(configFilePath string) (*Root, error) {
	file, err := os.Open(configFilePath)
	if err != nil {
		return nil, err
	}

	rootConfig := new(Root)

	decoder := candiedyaml.NewDecoder(file)
	err = decoder.Decode(rootConfig)
	if err != nil {
		return nil, err
	}

	err = validateConfig(*rootConfig)
	if err != nil {
		return nil, err
	}

	return rootConfig, nil
}

func validateConfig(rootConfig Root) error {
	rootConfigErr := validator.Validate(rootConfig)
	var errString string
	if rootConfigErr != nil {
		errString = formatErrorString(rootConfigErr, "")
	}

	for i, backend := range rootConfig.Proxy.Backends {
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

type Root struct {
	Proxy        Proxy `validate:"nonzero"`
	API          API   `validate:"nonzero"`
	ProfilerPort uint  `validate:"nonzero"`
}

type Proxy struct {
	Port                     uint      `validate:"nonzero"`
	Backends                 []Backend `validate:"min=1"`
	HealthcheckTimeoutMillis uint      `validate:"nonzero"`
}

type API struct {
	Port     uint   `validate:"nonzero"`
	Username string `validate:"nonzero"`
	Password string `validate:"nonzero"`
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
