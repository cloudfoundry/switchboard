package proxy_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSwitchboardProxy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Switchboard Proxy Suite")
}
