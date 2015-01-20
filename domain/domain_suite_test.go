package domain_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSwitchboardDomain(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Switchboard Domain Suite")
}
