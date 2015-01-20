package config_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestSwitchboarConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Switchboard Config Suite")
}
