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
	var fakeArpManager *fakes.FakeArpManager
	const healthcheckTimeout = time.Second

	BeforeEach(func() {
		fakeArpManager = nil
		backends = new(fakes.FakeBackends)
	})

	JustBeforeEach(func() {
		logger = lagertest.NewTestLogger("Cluster test")
		cluster = domain.NewCluster(backends, healthcheckTimeout, logger, fakeArpManager)
	})

	Describe("Monitor", func() {
		var backend1, backend2, backend3 *fakes.FakeBackend
		var urlGetter *fakes.FakeUrlGetter
		var healthyResponse = &http.Response{
			Body:       new(fakes.FakeReadWriteCloser),
			StatusCode: http.StatusOK,
		}

		BeforeEach(func() {
			backend1 = new(fakes.FakeBackend)
			backend1.AsJSONReturns(domain.BackendJSON{Host: "10.10.1.2"})
			backend1.HealthcheckUrlReturns("backend1")

			backend2 = new(fakes.FakeBackend)
			backend2.AsJSONReturns(domain.BackendJSON{Host: "10.10.2.2"})
			backend2.HealthcheckUrlReturns("backend2")

			backend3 = new(fakes.FakeBackend)
			backend3.AsJSONReturns(domain.BackendJSON{Host: "10.10.3.2"})
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

			urlGetter = new(fakes.FakeUrlGetter)
			urlGetter := urlGetter
			domain.UrlGetterProvider = func(time.Duration) domain.UrlGetter {
				return urlGetter
			}

			urlGetter.GetReturns(healthyResponse, nil)
		})

		AfterEach(func() {
			domain.UrlGetterProvider = domain.HttpUrlGetterProvider
		})

		It("notices when each backend stays healthy", func() {

			stopMonitoring := cluster.Monitor()
			defer close(stopMonitoring)

			Eventually(func() []interface{} {
				return getUniqueBackendArgs(
					backends.SetHealthyArgsForCall,
					backends.SetHealthyCallCount)
			}).Should(ConsistOf([]domain.Backend{
				backend1,
				backend2,
				backend3,
			}))
			Expect(backends.SetUnhealthyCallCount()).To(BeZero())
		})

		It("notices when a healthy backend becomes unhealthy", func() {

			unhealthyResponse := &http.Response{
				Body:       new(fakes.FakeReadWriteCloser),
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

			Eventually(func() []interface{} {
				return getUniqueBackendArgs(
					backends.SetHealthyArgsForCall,
					backends.SetHealthyCallCount)
			}).Should(ConsistOf([]domain.Backend{
				backend1,
				backend3,
			}))

			Eventually(backends.SetUnhealthyCallCount).Should(BeNumerically(">=", 1))
			Expect(backends.SetUnhealthyArgsForCall(0)).To(Equal(backend2))
		})

		It("notices when a healthy backend becomes unresponsive", func() {

			urlGetter.GetStub = func(url string) (*http.Response, error) {
				if url == "backend2" {
					return nil, errors.New("some error")
				} else {
					return healthyResponse, nil
				}
			}

			stopMonitoring := cluster.Monitor()
			defer close(stopMonitoring)

			Eventually(func() []interface{} {
				return getUniqueBackendArgs(
					backends.SetHealthyArgsForCall,
					backends.SetHealthyCallCount)
			}).Should(ConsistOf([]domain.Backend{
				backend1,
				backend3,
			}))

			Eventually(backends.SetUnhealthyCallCount).Should(BeNumerically(">=", 1))
			Expect(backends.SetUnhealthyArgsForCall(0)).To(Equal(backend2))
		})

		It("notices when an unhealthy backend becomes healthy", func() {
			unhealthyResponse := &http.Response{
				Body:       new(fakes.FakeReadWriteCloser),
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

			Eventually(backends.SetUnhealthyCallCount).Should(BeNumerically(">=", 1))
			Expect(backends.SetUnhealthyArgsForCall(0)).To(Equal(backend2))

			Eventually(func() []interface{} {
				return getUniqueBackendArgs(
					backends.SetHealthyArgsForCall,
					backends.SetHealthyCallCount)
			}).Should(ConsistOf([]domain.Backend{
				backend1,
				backend2,
				backend3,
			}))
		})

		Context("when a backend is healthy", func() {

			BeforeEach(func() {
				fakeArpManager = new(fakes.FakeArpManager)
			})

			It("does not clears arp cache after ArpFlushInterval has elapsed", func() {
				stopMonitoring := cluster.Monitor()
				defer close(stopMonitoring)

				Consistently(fakeArpManager.ClearCacheCallCount, healthcheckTimeout*2).Should(BeZero())
			})
		})

		Context("when a backend is unhealthy", func() {

			BeforeEach(func() {
				fakeArpManager = new(fakes.FakeArpManager)
				unhealthyResponse := &http.Response{
					Body:       new(fakes.FakeReadWriteCloser),
					StatusCode: http.StatusInternalServerError,
				}

				urlGetter.GetStub = func(url string) (*http.Response, error) {
					if url == "backend2" {
						return unhealthyResponse, nil
					} else {
						return healthyResponse, nil
					}
				}
			})

			Context("and the IP is in the ARP cache", func() {

				BeforeEach(func() {
					fakeArpManager.IsCachedStub = func(ip string) bool {
						if ip == backend2.AsJSON().Host {
							return true
						} else {
							return false
						}
					}
				})

				It("clears the arp cache after ArpFlushInterval has elapsed", func() {

					stopMonitoring := cluster.Monitor()
					defer close(stopMonitoring)

					Eventually(fakeArpManager.ClearCacheCallCount, 10*time.Second, 500*time.Millisecond).Should(BeNumerically(">=", 1), "Expected arpManager.ClearCache to be called at least once")
					Expect(fakeArpManager.ClearCacheArgsForCall(0)).To(Equal(backend2.AsJSON().Host))
				})
			})

			Context("and the IP is not in the ARP cache", func() {

				BeforeEach(func() {
					fakeArpManager.IsCachedReturns(false)
				})

				It("does not clear arp cache", func() {

					stopMonitoring := cluster.Monitor()
					defer close(stopMonitoring)

					Consistently(fakeArpManager.ClearCacheCallCount, healthcheckTimeout*2).Should(BeZero())
				})
			})
		})
	})

	Describe("RouteToBackend", func() {
		var clientConn net.Conn

		BeforeEach(func() {
			clientConn = new(fakes.FakeConn)
		})

		It("bridges the client connection to the active backend", func() {
			activeBackend := new(fakes.FakeBackend)
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

func getUniqueBackendArgs(getArgsForCall func(int) domain.Backend, getCallCount func() int) []interface{} {

	args := []interface{}{}
	backendMap := make(map[string]bool)
	callCount := getCallCount()
	for i := 0; i < callCount; i++ {
		arg := getArgsForCall(i)
		host := arg.AsJSON().Host
		if _, keyExists := backendMap[host]; keyExists == false {
			args = append(args, arg)
			backendMap[host] = true
		}
	}

	return args
}
