package config

import (
	"os"
	"time"

	"github.com/fraenkel/candiedyaml"
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

	return rootConfig, nil
}

type Root struct {
	Proxy Proxy
	API   API
}

type Proxy struct {
	Port                     uint
	Backends                 []Backend
	HealthcheckTimeoutMillis uint
}

type API struct {
	Port     uint
	Username string
	Password string
}

type Backend struct {
	BackendHost     string
	BackendPort     uint
	HealthcheckPort uint
	BackendName     string
}

func (p Proxy) HealthcheckTimeout() time.Duration {
	return time.Duration(p.HealthcheckTimeoutMillis) * time.Millisecond
}
