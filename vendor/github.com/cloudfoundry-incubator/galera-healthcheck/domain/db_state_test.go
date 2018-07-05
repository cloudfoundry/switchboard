package domain_test

import (
	"github.com/cloudfoundry-incubator/galera-healthcheck/domain"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("WsrepLocalState", func() {
	DescribeTable("Comment",
		func(state domain.WsrepLocalState, comment domain.WsrepLocalStateComment) {
			Expect(state.Comment()).To(Equal(comment))
		},
		Entry("maps joining", domain.Joining, domain.JoiningString),
		Entry("maps donor desynced", domain.DonorDesynced, domain.DonorDesyncedString),
		Entry("maps joined", domain.Joined, domain.JoinedString),
		Entry("maps synced", domain.Synced, domain.SyncedString),
		Entry("maps unknown value of 0", domain.WsrepLocalState(0), domain.WsrepLocalStateComment("Unrecognized state: 0")),
		Entry("maps unknown value greater than 4", domain.WsrepLocalState(1234), domain.WsrepLocalStateComment("Unrecognized state: 1234")),
	)
})
