package switchboard_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSwitchboard(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Switchboard Library Suite")
}
