package monitor_test

import (
	"errors"
	"net"

	. "github.com/cloudfoundry-incubator/switchboard/runner/monitor"
	"github.com/cloudfoundry-incubator/switchboard/runner/monitor/monitorfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ARPFlusher", func() {

	var (
		runner *monitorfakes.FakeCmdRunner
		arp    ArpEntryRemover
	)

	BeforeEach(func() {
		runner = new(monitorfakes.FakeCmdRunner)
		arp = NewARPFlusher(runner)
	})

	Describe("RemoveEntry", func() {
		It("runs", func() {
			err := arp.RemoveEntry(net.ParseIP("192.0.2.0"))
			Expect(err).ToNot(HaveOccurred())

			Expect(runner.RunCallCount()).To(Equal(1))

			cmd, args := runner.RunArgsForCall(0)
			Expect(cmd).To(Equal(FlushARPBinPath))
			Expect(args).To(Equal([]string{"192.0.2.0"}))
		})

		Context("when there is an error", func() {
			var (
				output      string
				expectedErr error
			)

			BeforeEach(func() {
				expectedErr = errors.New("some error")
				output = "some output"

				runner.RunReturns([]byte(output), expectedErr)
			})

			It("returns an error", func() {

				err := arp.RemoveEntry(net.ParseIP("192.0.2.0"))
				Expect(err.Error()).To(ContainSubstring(expectedErr.Error()))
				Expect(err.Error()).To(ContainSubstring(output))

				Expect(runner.RunCallCount()).To(Equal(1))

				cmd, args := runner.RunArgsForCall(0)
				Expect(cmd).To(Equal(FlushARPBinPath))
				Expect(args).To(Equal([]string{"192.0.2.0"}))
			})
		})
	})
})
