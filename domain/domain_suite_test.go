package domain_test

import (
	"testing"
  "time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSwitchboardDomain(t *testing.T) {

  // this suite involves a high amount of concurrency
  // setting the timeout higher reduces flakiness on systems
  // with fewer cores
  SetDefaultEventuallyTimeout(5*time.Second)
  SetDefaultEventuallyPollingInterval(500*time.Millisecond)

	RegisterFailHandler(Fail)
	RunSpecs(t, "Switchboard Domain Suite")
}
