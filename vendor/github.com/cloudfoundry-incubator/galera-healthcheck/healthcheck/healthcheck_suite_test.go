package healthcheck_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestGalera_healthcheck(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "HealthCheck Suite")
}
