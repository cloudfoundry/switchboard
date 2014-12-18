package config

import (
	"os"
	"time"

	"github.com/fraenkel/candiedyaml"
)

func Load(configFilePath string) (*Proxy, error) {
	file, err := os.Open(configFilePath)
	if err != nil {
		return nil, err
	}

	proxyConfig := new(Proxy)

	decoder := candiedyaml.NewDecoder(file)
	err = decoder.Decode(proxyConfig)
	if err != nil {
		return nil, err
	}

	return proxyConfig, nil
}

type Proxy struct {
	Port               uint
	Pidfile            string
	Backends           []Backend
	HealthcheckTimeout time.Duration
}

type Backend struct {
	BackendIP       string
	BackendPort     uint
	HealthcheckPort uint
}
