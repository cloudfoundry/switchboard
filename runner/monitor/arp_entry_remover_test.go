package monitor_test

import (
	"errors"
	"net"

	. "github.com/cloudfoundry-incubator/switchboard/runner/monitor"
	"github.com/cloudfoundry-incubator/switchboard/runner/monitor/monitorfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ArpEntryRemover", func() {

	var (
		runner *monitorfakes.FakeCmdRunner
		arp    ArpEntryRemover
	)

	BeforeEach(func() {
		runner = new(monitorfakes.FakeCmdRunner)
	})

	Describe("RemoveEntry", func() {
		It("deletes the entry", func() {
			runner.RunReturns([]byte{}, nil)
			arp = NewPrivilegedArpEntryRemover(runner)
			err := arp.RemoveEntry(net.ParseIP("192.0.2.0"))
			Expect(err).ToNot(HaveOccurred())

			Expect(runner.RunCallCount()).To(Equal(1))
			cmd, args := runner.RunArgsForCall(0)
			Expect(cmd).To(Equal("/usr/sbin/arp"))
			Expect(args).To(Equal([]string{"-d", "192.0.2.0"}))
		})
		Context("when the entry cannot be deleted", func() {
			Context("when the ip is in a wrong format", func() {
				It("returns an error", func() {
					arp = NewPrivilegedArpEntryRemover(runner)
					err := arp.RemoveEntry(net.ParseIP("invalidIP"))
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError("failed to delete arp entry: invalid IP"))
				})
			})
			It("returns an error", func() {
				runner.RunReturns(
					[]byte("SIOCDARP(dontpub): Operation not permitted"),
					errors.New("exit status 255"))
				arp = NewPrivilegedArpEntryRemover(runner)
				err := arp.RemoveEntry(net.ParseIP("192.0.2.0"))
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("failed to delete arp entry: OUTPUT=SIOCDARP(dontpub): " +
					"Operation not permitted, ERROR=exit status 255"))

				Expect(runner.RunCallCount()).To(Equal(1))
				cmd, args := runner.RunArgsForCall(0)
				Expect(cmd).To(Equal("/usr/sbin/arp"))
				Expect(args).To(Equal([]string{"-d", "192.0.2.0"}))
			})
		})
	})
})
