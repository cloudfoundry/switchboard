package middleware_test

import (
	"net/http"

	apifakes "github.com/cloudfoundry-incubator/switchboard/api/fakes"

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
	var fakeLogger *fakes.FakeLogger

	BeforeEach(func() {
		dummyRequest, err = http.NewRequest("GET", "/v0/backends", nil)
		Expect(err).NotTo(HaveOccurred())
		dummyRequest.Header.Add("Authorization", "some auth")

		fakeResponseWriter = &apifakes.FakeResponseWriter{}
		fakeHandler = &fakes.FakeHandler{}
		fakeLogger = &fakes.FakeLogger{}
	})

	It("should write to logger", func() {
		loggerMiddleware := middleware.NewLogger(fakeLogger)
		loggerHandler := loggerMiddleware.Wrap(fakeHandler)

		loggerHandler.ServeHTTP(fakeResponseWriter, dummyRequest)

		Expect(fakeLogger.InfoCallCount()).To(Equal(1))
		arg0, _ := fakeLogger.InfoArgsForCall(0)
		Expect(arg0).To(ContainSubstring("GET"))
	})

	It("should not log credentials", func() {
		loggerMiddleware := middleware.NewLogger(fakeLogger)
		loggerHandler := loggerMiddleware.Wrap(fakeHandler)

		loggerHandler.ServeHTTP(fakeResponseWriter, dummyRequest)

		Expect(fakeLogger.InfoCallCount()).To(Equal(1))
		_, arg1 := fakeLogger.InfoArgsForCall(0)
		loggedRequest := arg1[0]["request"].(http.Request)
		Expect(loggedRequest.BasicAuth()).To(Equal(""))
	})

	It("should call next handler", func() {
		loggerMiddleware := middleware.NewLogger(fakeLogger)
		loggerHandler := loggerMiddleware.Wrap(fakeHandler)

		loggerHandler.ServeHTTP(fakeResponseWriter, dummyRequest)

		Expect(fakeHandler.ServeHTTPCallCount()).To(Equal(1))
		arg0, arg1 := fakeHandler.ServeHTTPArgsForCall(0)
		Expect(arg0).ToNot(BeNil())
		Expect(arg1).To(Equal(dummyRequest))
	})
})
