package domain_test

import (
	"errors"
	"net"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/switchboard/domain"
	"github.com/cloudfoundry-incubator/switchboard/domain/fakes"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("Cluster", func() {
	var backends *fakes.FakeBackends
	var logger lager.Logger
	var cluster domain.Cluster

	BeforeEach(func() {
		backends = &fakes.FakeBackends{}
		logger = lagertest.NewTestLogger("Cluster test")
		cluster = domain.NewCluster(backends, time.Second, logger)
	})

	Describe("Monitor", func() {
		var backend1, backend2, backend3 *fakes.FakeBackend
		var urlGetter *fakes.FakeUrlGetter
		var healthyResponse = &http.Response{
			Body:       &fakes.FakeReadWriteCloser{},
			StatusCode: http.StatusOK,
		}

		BeforeEach(func() {
			backend1 = &fakes.FakeBackend{}
			backend1.HealthcheckUrlReturns("backend1")

			backend2 = &fakes.FakeBackend{}
			backend2.HealthcheckUrlReturns("backend2")

			backend3 = &fakes.FakeBackend{}
			backend3.HealthcheckUrlReturns("backend3")

			backends.AllStub = func() <-chan domain.Backend {
				c := make(chan domain.Backend)
				go func() {
					c <- backend1
					c <- backend2
					c <- backend3
					close(c)
				}()
				return c
			}

			urlGetter = &fakes.FakeUrlGetter{}
			urlGetter := urlGetter
			domain.UrlGetterProvider = func(time.Duration) domain.UrlGetter {
				return urlGetter
			}

			urlGetter.GetReturns(healthyResponse, nil)
		})

		AfterEach(func() {
			domain.UrlGetterProvider = domain.HttpUrlGetterProvider
		})

		It("notices when each backend stays healthy", func(done Done) {
			defer close(done)

			stopMonitoring := cluster.Monitor()
			defer close(stopMonitoring)

			Eventually(backends.SetHealthyCallCount, 2*time.Second).Should(BeNumerically(">=", 3))
			Expect(backends.SetHealthyArgsForCall(0)).To(Equal(backend1))
			Expect(backends.SetHealthyArgsForCall(1)).To(Equal(backend2))
			Expect(backends.SetHealthyArgsForCall(2)).To(Equal(backend3))
		}, 5)

		It("notices when a healthy backend becomes unhealthy", func(done Done) {
			defer close(done)

			unhealthyResponse := &http.Response{
				Body:       &fakes.FakeReadWriteCloser{},
				StatusCode: http.StatusInternalServerError,
			}

			urlGetter.GetStub = func(url string) (*http.Response, error) {
				if url == "backend2" {
					return unhealthyResponse, nil
				} else {
					return healthyResponse, nil
				}
			}

			stopMonitoring := cluster.Monitor()
			defer close(stopMonitoring)

			Eventually(backends.SetHealthyCallCount, 2*time.Second).Should(BeNumerically(">=", 2))
			Expect(backends.SetHealthyArgsForCall(0)).To(Equal(backend1))
			Expect(backends.SetHealthyArgsForCall(1)).To(Equal(backend3))

			Expect(backends.SetUnhealthyCallCount()).To(BeNumerically(">=", 1))
			Expect(backends.SetUnhealthyArgsForCall(0)).To(Equal(backend2))
		}, 5)

		It("notices when a healthy backend becomes unresponsive", func(done Done) {
			defer close(done)

			urlGetter.GetStub = func(url string) (*http.Response, error) {
				if url == "backend2" {
					return nil, errors.New("some error")
				} else {
					return healthyResponse, nil
				}
			}

			stopMonitoring := cluster.Monitor()
			defer close(stopMonitoring)

			Eventually(backends.SetHealthyCallCount, 2*time.Second).Should(BeNumerically(">=", 2))
			Expect(backends.SetHealthyArgsForCall(0)).Should(Equal(backend1))
			Expect(backends.SetHealthyArgsForCall(1)).Should(Equal(backend3))

			Expect(backends.SetUnhealthyCallCount()).Should(BeNumerically(">=", 1))
			Expect(backends.SetUnhealthyArgsForCall(0)).Should(Equal(backend2))
		}, 5)

		It("notices when an unhealthy backend becomes healthy", func(done Done) {
			defer close(done)

			unhealthyResponse := &http.Response{
				Body:       &fakes.FakeReadWriteCloser{},
				StatusCode: http.StatusInternalServerError,
			}

			isUnhealthy := true
			urlGetter.GetStub = func(url string) (*http.Response, error) {
				if url == "backend2" && isUnhealthy {
					isUnhealthy = false
					return unhealthyResponse, nil
				} else {
					return healthyResponse, nil
				}
			}

			stopMonitoring := cluster.Monitor()
			defer close(stopMonitoring)

			initialHealthyBackendCount := 2
			initialUnhealthyBackendCount := 1
			finalHealthyBackendCount := 3
			Eventually(backends.SetHealthyCallCount, 2*time.Second).Should(BeNumerically(">=", initialHealthyBackendCount+finalHealthyBackendCount))
			Expect(backends.SetUnhealthyCallCount()).To(Equal(initialUnhealthyBackendCount))
		}, 5)
	})

	Describe("RouteToBackend", func() {
		var clientConn net.Conn

		BeforeEach(func() {
			clientConn = &fakes.FakeConn{}
		})

		It("bridges the client connection to the active backend", func() {
			activeBackend := &fakes.FakeBackend{}
			backends.ActiveReturns(activeBackend)

			err := cluster.RouteToBackend(clientConn)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(activeBackend.BridgeCallCount()).To(Equal(1))
			Expect(activeBackend.BridgeArgsForCall(0)).To(Equal(clientConn))
		})

		It("returns an error if there is no active backend", func() {
			backends.ActiveReturns(nil)

			err := cluster.RouteToBackend(clientConn)

			Expect(err).Should(HaveOccurred())
		})
	})
})
