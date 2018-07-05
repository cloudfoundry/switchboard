package main_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestGaleraHealthcheck(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Galera Healthcheck Server Suite")
}
