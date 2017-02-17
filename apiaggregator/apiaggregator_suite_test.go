package apiaggregator_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSwitchboardAPIAggregator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Switchboard API Aggregator Suite")
}
