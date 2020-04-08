package config

import (
	"errors"
	"flag"
	"fmt"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagerflags"
	"github.com/pivotal-cf-experimental/service-config"
	"gopkg.in/validator.v2"

	"github.com/cloudfoundry-incubator/galera-healthcheck/domain"
)

type Config struct {
	DB                    DBConfig    `yaml:"DB" validate:"nonzero"`
	Monit                 MonitConfig `yaml:"Monit" validate:"nonzero"`
	Host                  string      `yaml:"Host" validate:"nonzero"`
	Port                  int         `yaml:"Port" validate:"nonzero"`
	AvailableWhenDonor    bool        `yaml:"AvailableWhenDonor"`
	AvailableWhenReadOnly bool        `yaml:"AvailableWhenReadOnly"`
	Logger                lager.Logger
	MysqldPath            string                `yaml:"MysqldPath" validate:"nonzero"`
	MyCnfPath             string                `yaml:"MyCnfPath" validate:"nonzero"`
	SidecarEndpoint       SidecarEndpointConfig `yaml:"SidecarEndpoint" validate:"nonzero"`
}

type DBConfig struct {
	Host     string `yaml:"Host" validate:"nonzero"`
	User     string `yaml:"User" validate:"nonzero"`
	Port     int    `yaml:"Port" validate:"nonzero"`
	Password string `yaml:"Password" validate:"nonzero"`
}

type MonitConfig struct {
	Host                          string `yaml:"Host" validate:"nonzero"`
	User                          string `yaml:"User" validate:"nonzero"`
	Port                          string `yaml:"Port" validate:"nonzero"`
	Password                      string `yaml:"Password" validate:"nonzero"`
	MysqlStateFilePath            string `yaml:"MysqlStateFilePath"`
	ServiceName                   string `yaml:"ServiceName" validate:"nonzero"`
	GaleraInitStatusServerAddress string `yaml:"GaleraInitStatusServerAddress" validate:"nonzero"`
}

type SidecarEndpointConfig struct {
	Username string `yaml:"Username" validate:"nonzero"`
	Password string `yaml:"Password" validate:"nonzero"`
}

func defaultConfig() *Config {
	var defaultConfig Config
	defaultConfig = Config{
		Host: "0.0.0.0",
		Port: 8080,
		DB: DBConfig{
			Host:     "0.0.0.0",
			Port:     3306,
			User:     "root",
			Password: "",
		},
		AvailableWhenDonor:    true,
		AvailableWhenReadOnly: false,
	}
	return &defaultConfig
}

func NewConfig(osArgs []string) (*Config, error) {
	var rootConfig Config

	binaryName := osArgs[0]
	configurationOptions := osArgs[1:]
	serviceConfig := service_config.New()
	flags := flag.NewFlagSet(binaryName, flag.ExitOnError)

	lagerflags.AddFlags(flags)

	serviceConfig.AddFlags(flags)
	serviceConfig.AddDefaults(defaultConfig())
	flags.Parse(configurationOptions)

	rootConfig.Logger, _ = lagerflags.NewFromConfig(binaryName, lagerflags.ConfigFromFlags())

	err := serviceConfig.Read(&rootConfig)
	return &rootConfig, err
}

func (c Config) Validate() error {
	rootConfigErr := validator.Validate(c)
	var errString string
	if rootConfigErr != nil {
		errString = formatErrorString(rootConfigErr, "")
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

func (c *Config) IsHealthy(state domain.DBState) bool {
	if state.ReadOnly && !c.AvailableWhenReadOnly {
		return false
	}

	return (domain.WsrepLocalState(state.WsrepLocalState) == domain.Synced) || (domain.WsrepLocalState(state.WsrepLocalState) == domain.DonorDesynced && c.AvailableWhenDonor)
}
