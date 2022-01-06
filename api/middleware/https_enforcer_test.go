package middleware_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/cloudfoundry-incubator/switchboard/api/middleware"
	"github.com/cloudfoundry-incubator/switchboard/api/middleware/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("HttpsEnforcer", func() {
	var (
		request           *http.Request
		writer            *httptest.ResponseRecorder
		fakeHandler       *fakes.FakeHandler
		wrappedMiddleware http.Handler
		forceHttps        bool
	)

	BeforeEach(func() {
		forceHttps = true
	})

	JustBeforeEach(func() {
		fakeHandler = new(fakes.FakeHandler)
		writer = httptest.NewRecorder()
		enforcer := middleware.NewHttpsEnforcer(forceHttps)

		wrappedMiddleware = enforcer.Wrap(fakeHandler)
	})

	Context("With https header", func() {
		BeforeEach(func() {
			request, _ = http.NewRequest("GET", "https://localhost/foo/bar", nil)
			request.Header.Set("X-Forwarded-Proto", "https")
		})

		It("calls next middleware", func() {
			wrappedMiddleware.ServeHTTP(writer, request)

			Expect(fakeHandler.ServeHTTPCallCount()).To(Equal(1))
		})
	})

	Context("With chained https header", func() {
		BeforeEach(func() {
			request, _ = http.NewRequest("GET", "https://localhost/foo/bar", nil)
			request.Header.Set("X-Forwarded-Proto", "https,https")
		})

		It("calls next middleware", func() {
			wrappedMiddleware.ServeHTTP(writer, request)

			Expect(fakeHandler.ServeHTTPCallCount()).To(Equal(1))
		})
		When("with leading whitespace between header values", func() {
			BeforeEach(func() {
				request, _ = http.NewRequest("GET", "https://localhost/foo/bar", nil)
				request.Header.Set("X-Forwarded-Proto", "https, https")
			})
			It("calls next middleware", func() {
				wrappedMiddleware.ServeHTTP(writer, request)

				Expect(fakeHandler.ServeHTTPCallCount()).To(Equal(1))
			})
		})
	})

	Context("With mixed https, http header", func() {
		BeforeEach(func() {
			request, _ = http.NewRequest("GET", "https://localhost/foo/bar", nil)
			request.Header.Set("X-Forwarded-Proto", "http,https")
		})

		It("does not call next middleware", func() {
			wrappedMiddleware.ServeHTTP(writer, request)

			Expect(fakeHandler.ServeHTTPCallCount()).To(BeZero())
		})
	})

	Context("With a http-only header", func() {
		BeforeEach(func() {
			request, _ = http.NewRequest("GET", "http://localhost/foo/bar", nil)
			request.Header.Set("X-Forwarded-Proto", "http")
		})

		It("does not call next middleware", func() {
			wrappedMiddleware.ServeHTTP(writer, request)

			Expect(fakeHandler.ServeHTTPCallCount()).To(BeZero())
		})

		It("redirects to https", func() {
			wrappedMiddleware.ServeHTTP(writer, request)

			Expect(writer.Code).To(Equal(http.StatusFound))
			Expect(writer.HeaderMap.Get("Location")).To(Equal("https://localhost/foo/bar"))
		})

		Context("when ForceHttps is false", func() {
			BeforeEach(func() {
				forceHttps = false
			})

			It("calls the next middleware", func() {
				wrappedMiddleware.ServeHTTP(writer, request)

				Expect(fakeHandler.ServeHTTPCallCount()).To(Equal(1))
			})
		})
	})

	// TODO: Reactor above & below to re-use copy/pasted assertions
	Context("With an blank X-Forwarded-Proto header", func() {
		BeforeEach(func() {
			request, _ = http.NewRequest("GET", "http://localhost/foo/bar", nil)
			request.Header.Set("X-Forwarded-Proto", "")
		})
		It("does not call next middleware", func() {
			wrappedMiddleware.ServeHTTP(writer, request)

			Expect(fakeHandler.ServeHTTPCallCount()).To(BeZero())
		})

		It("redirects to https", func() {
			wrappedMiddleware.ServeHTTP(writer, request)

			Expect(writer.Code).To(Equal(http.StatusFound))
			Expect(writer.HeaderMap.Get("Location")).To(Equal("https://localhost/foo/bar"))
		})
	})

	Context("With an non-existent X-Forwarded-Proto header", func() {
		BeforeEach(func() {
			request, _ = http.NewRequest("GET", "http://localhost/foo/bar", nil)
		})
		It("does not call next middleware", func() {
			wrappedMiddleware.ServeHTTP(writer, request)

			Expect(fakeHandler.ServeHTTPCallCount()).To(BeZero())
		})

		It("redirects to https", func() {
			wrappedMiddleware.ServeHTTP(writer, request)

			Expect(writer.Code).To(Equal(http.StatusFound))
			Expect(writer.HeaderMap.Get("Location")).To(Equal("https://localhost/foo/bar"))
		})

	})
})
