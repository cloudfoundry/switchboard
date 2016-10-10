package middleware_test

import (
	"net/http"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	"github.com/cloudfoundry-incubator/switchboard/api/apifakes"

	"github.com/cloudfoundry-incubator/switchboard/api/middleware"
	"github.com/cloudfoundry-incubator/switchboard/api/middleware/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Logger", func() {

	var dummyRequest *http.Request
	var err error

	var fakeResponseWriter http.ResponseWriter
	var fakeHandler *fakes.FakeHandler
	var logger *lagertest.TestLogger
	var routePrefix string

	const fakePassword = "fakePassword"

	BeforeEach(func() {
		routePrefix = "/v0"
		dummyRequest, err = http.NewRequest("GET", "/v0/backends", nil)
		Expect(err).NotTo(HaveOccurred())
		dummyRequest.Header.Add("Authorization", fakePassword)

		fakeResponseWriter = new(apifakes.FakeResponseWriter)
		fakeHandler = new(fakes.FakeHandler)

		logger = lagertest.NewTestLogger("backup-download-test")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.INFO))
	})

	It("should log requests that are prefixed with routePrefix", func() {
		loggerMiddleware := middleware.NewLogger(logger, routePrefix)
		loggerHandler := loggerMiddleware.Wrap(fakeHandler)

		loggerHandler.ServeHTTP(fakeResponseWriter, dummyRequest)

		logContents := logger.Buffer().Contents()
		Expect(logContents).To(ContainSubstring("request"))
		Expect(logContents).To(ContainSubstring("response"))
	})

	It("should not log credentials", func() {
		loggerMiddleware := middleware.NewLogger(logger, routePrefix)
		loggerHandler := loggerMiddleware.Wrap(fakeHandler)

		loggerHandler.ServeHTTP(fakeResponseWriter, dummyRequest)

		logContents := logger.Buffer().Contents()
		Expect(logContents).ToNot(ContainSubstring(fakePassword))
	})

	It("should call next handler", func() {
		loggerMiddleware := middleware.NewLogger(logger, routePrefix)
		loggerHandler := loggerMiddleware.Wrap(fakeHandler)

		loggerHandler.ServeHTTP(fakeResponseWriter, dummyRequest)

		Expect(fakeHandler.ServeHTTPCallCount()).To(Equal(1))
		arg0, arg1 := fakeHandler.ServeHTTPArgsForCall(0)
		Expect(arg0).ToNot(BeNil())
		Expect(arg1).To(Equal(dummyRequest))
	})
})
