package api_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestBootstrapAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bootstrap API Suite")
}
