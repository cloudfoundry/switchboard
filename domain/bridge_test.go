package domain_test

import (
	"errors"
	"io"

	"github.com/cloudfoundry-incubator/switchboard/domain"
	"github.com/cloudfoundry-incubator/switchboard/domain/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("Bridge", func() {
	Describe("#Connect", func() {
		var bridge domain.Bridge
		var client, backend *fakes.FakeConn
		var logger lager.Logger

		BeforeEach(func() {
			logger = lagertest.NewTestLogger("Bridge test")
			backend = &fakes.FakeConn{}
			client = &fakes.FakeConn{}

			clientAddr := &fakes.FakeAddr{}
			backendAddr := &fakes.FakeAddr{}

			client.RemoteAddrReturns(clientAddr)
			backend.RemoteAddrReturns(backendAddr)

			bridge = domain.NewBridge(client, backend, logger)
		})

		Context("When operating normally", func() {

			It("forwards data from the client to backend", func() {
				expectedText := "hello"
				var copiedToBackend string
				clientReadCount := 0
				client.ReadStub = func(p []byte) (int, error) {
					if clientReadCount == 0 {
						copy(p, expectedText)
						clientReadCount++
						return len(expectedText), nil
					}
					return 0, io.EOF
				}

				backend.WriteStub = func(p []byte) (int, error) {
					copiedToBackend = string(p)
					return len(expectedText), nil
				}

				go bridge.Connect()
				defer bridge.Close()
				Eventually(client.ReadCallCount).Should(Equal(2))
				Eventually(backend.WriteCallCount).Should(Equal(1))
				Expect(copiedToBackend).To(Equal(expectedText))
			})

			It("forwards data from the backend to client", func() {
				expectedText := "echo: hello"
				var copiedToClient string

				backendReadCount := 0
				backend.ReadStub = func(p []byte) (int, error) {
					if backendReadCount == 0 {
						copy(p, expectedText)
						backendReadCount++
						return len(expectedText), nil
					}
					return 0, io.EOF
				}

				client.WriteStub = func(p []byte) (int, error) {
					copiedToClient = string(p)
					return len(expectedText), nil
				}

				go bridge.Connect()
				defer bridge.Close()
				Eventually(backend.ReadCallCount).Should(Equal(2))
				Eventually(client.WriteCallCount).Should(Equal(1))
				Expect(copiedToClient).To(Equal(expectedText))
			})
		})

		Context("when the client returns an error", func() {
			BeforeEach(func() {
				client.ReadReturns(0, errors.New("Error reading from client"))
			})

			It("Closes the backend", func() {
				bridge.Connect()
				Expect(backend.CloseCallCount()).To(Equal(1))
			})
		})

		Context("when the client returns EOF", func() {
			BeforeEach(func() {
				client.ReadStub = func(p []byte) (int, error) {
					return 0, io.EOF
				}
			})

			It("Closes the backend", func() {
				bridge.Connect()
				Expect(backend.CloseCallCount()).To(Equal(1))
			})
		})

		Context("when the backend returns an error", func() {
			BeforeEach(func() {
				backend.ReadReturns(0, errors.New("Error reading from backend"))
			})

			It("Closes the client", func() {
				bridge.Connect()
				Expect(client.CloseCallCount()).To(Equal(1))
			})
		})

		Context("when the backend returns EOF", func() {
			BeforeEach(func() {
				backend.ReadStub = func(p []byte) (int, error) {
					return 0, io.EOF
				}
			})

			It("Closes the client", func() {
				bridge.Connect()
				Expect(client.CloseCallCount()).To(Equal(1))
			})
		})

		Context("When the connection is closed by calling Close()", func() {
			It("Closes the client and backend", func() {
				go bridge.Connect()
				bridge.Close()
				Eventually(backend.CloseCallCount).Should(Equal(1))
				Eventually(client.CloseCallCount).Should(Equal(1))
			})
		})
	})
})
