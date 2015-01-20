package api_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSwitchboardAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Switchboard API Suite")
}
