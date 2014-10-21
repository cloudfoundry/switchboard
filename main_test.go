package main_test

import (
	"fmt"
	"net"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

func startMainWithArgs(args ...string) *gexec.Session {
	command := exec.Command(switchboardBinPath, args...)
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session).Should(gbytes.Say("started on port"))
	return session
}

var _ = Describe("Switchboard", func() {

	It("accepts multiple tcp connections on specified port", func() {

		args := []string{
			fmt.Sprintf("-port=%d", switchboardPort),
		}

		session := startMainWithArgs(args...)
		defer session.Terminate()

		Consistently(func() error {
			_, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
			_, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
			return err
		}, 1*time.Second, 10*time.Millisecond).ShouldNot(HaveOccurred())
	})
})
