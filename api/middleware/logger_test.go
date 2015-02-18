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
	var routePrefix string

	BeforeEach(func() {
		routePrefix = "/v0"
		dummyRequest, err = http.NewRequest("GET", "/v0/backends", nil)
		Expect(err).NotTo(HaveOccurred())
		dummyRequest.Header.Add("Authorization", "some auth")

		fakeResponseWriter = &apifakes.FakeResponseWriter{}
		fakeHandler = &fakes.FakeHandler{}
		fakeLogger = &fakes.FakeLogger{}
	})

	It("should log requests that are prefixed with routePrefix", func() {
		loggerMiddleware := middleware.NewLogger(fakeLogger, routePrefix)
		loggerHandler := loggerMiddleware.Wrap(fakeHandler)

		loggerHandler.ServeHTTP(fakeResponseWriter, dummyRequest)

		Expect(fakeLogger.InfoCallCount()).To(Equal(1))
		_, arg1 := fakeLogger.InfoArgsForCall(0)
		lagerData := arg1[0]
		Expect(lagerData["request"]).NotTo(BeNil())
		Expect(lagerData["response"]).NotTo(BeNil())
	})

	It("should not log requests that are not prefixed with routePrefix", func() {
		dummyRequest, err = http.NewRequest("GET", "/", nil)
		Expect(err).NotTo(HaveOccurred())
		dummyRequest.Header.Add("Authorization", "some auth")

		loggerMiddleware := middleware.NewLogger(fakeLogger, routePrefix)
		loggerHandler := loggerMiddleware.Wrap(fakeHandler)

		loggerHandler.ServeHTTP(fakeResponseWriter, dummyRequest)

		Expect(fakeLogger.InfoCallCount()).To(Equal(0))
	})

	It("should not log credentials", func() {
		loggerMiddleware := middleware.NewLogger(fakeLogger, routePrefix)
		loggerHandler := loggerMiddleware.Wrap(fakeHandler)

		loggerHandler.ServeHTTP(fakeResponseWriter, dummyRequest)

		Expect(fakeLogger.InfoCallCount()).To(Equal(1))
		_, arg1 := fakeLogger.InfoArgsForCall(0)
		loggedRequest := arg1[0]["request"].(http.Request)
		Expect(loggedRequest.BasicAuth()).To(Equal(""))
	})

	It("should call next handler", func() {
		loggerMiddleware := middleware.NewLogger(fakeLogger, routePrefix)
		loggerHandler := loggerMiddleware.Wrap(fakeHandler)

		loggerHandler.ServeHTTP(fakeResponseWriter, dummyRequest)

		Expect(fakeHandler.ServeHTTPCallCount()).To(Equal(1))
		arg0, arg1 := fakeHandler.ServeHTTPArgsForCall(0)
		Expect(arg0).ToNot(BeNil())
		Expect(arg1).To(Equal(dummyRequest))
	})
})
