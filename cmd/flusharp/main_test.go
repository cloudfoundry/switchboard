package main_test

import (
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("ARP Flusher", func() {
	var (
		binaryPath string
	)

	BeforeEach(func() {
		var err error
		binaryPath, err = gexec.Build("github.com/cloudfoundry-incubator/switchboard/cmd/flusharp")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		gexec.CleanupBuildArtifacts()
	})

	It("raises an error if an argument is not provided", func() {
		command := exec.Command(
			binaryPath,
		)

		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		Eventually(session).Should(gexec.Exit())
		Expect(session.ExitCode()).NotTo(BeZero())

		stderr := string(session.Err.Contents())
		Expect(stderr).To(ContainSubstring("arguments"))
	})

	It("raises an error if the ip is invalid", func() {
		command := exec.Command(
			binaryPath,
			"invalid-ip",
		)

		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		Eventually(session).Should(gexec.Exit())
		Expect(session.ExitCode()).NotTo(BeZero())

		stderr := string(session.Err.Contents())
		Expect(stderr).To(ContainSubstring("invalid"))
	})

	It("raises an error when ARP flushing fails", func() {
		command := exec.Command(
			binaryPath,
			"192.0.2.1",
		)

		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		Eventually(session).Should(gexec.Exit())
		Expect(session.ExitCode()).NotTo(BeZero())

		stderr := string(session.Err.Contents())
		Expect(stderr).To(ContainSubstring("exit status"))
	})
})
