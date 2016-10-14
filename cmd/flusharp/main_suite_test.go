package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestARPFlusher(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ARP Flusher Suite")
}
