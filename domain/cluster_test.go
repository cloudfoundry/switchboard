package domain_test

import (
	"errors"
	"net"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"bytes"
	"io/ioutil"

	"github.com/cloudfoundry-incubator/switchboard/domain"
	"github.com/cloudfoundry-incubator/switchboard/domain/domainfakes"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"
)

const healthcheckTimeout = time.Second

var _ = Describe("Cluster", func() {
	var (
		backends                     *domainfakes.FakeBackends
		backendSlice                 []*domainfakes.FakeBackend
		logger                       lager.Logger
		cluster                      *domain.Cluster
		fakeArpManager               *domainfakes.FakeArpManager
		backend1, backend2, backend3 *domainfakes.FakeBackend
	)

	BeforeEach(func() {
		fakeArpManager = nil
		backends = new(domainfakes.FakeBackends)

		backend1 = new(domainfakes.FakeBackend)
		backend1.AsJSONReturns(domain.BackendJSON{Host: "10.10.1.2"})
		backend1.HealthcheckUrlReturns("backend1")

		backend2 = new(domainfakes.FakeBackend)
		backend2.AsJSONReturns(domain.BackendJSON{Host: "10.10.2.2"})
		backend2.HealthcheckUrlReturns("backend2")
		backend2.TrafficEnabledReturns(true)

		backend3 = new(domainfakes.FakeBackend)
		backend3.AsJSONReturns(domain.BackendJSON{Host: "10.10.3.2"})
		backend3.HealthcheckUrlReturns("backend3")
		backend3.TrafficEnabledReturns(true)

		backendSlice = []*domainfakes.FakeBackend{backend1, backend2, backend3}

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

		backends.AnyReturns(backend1)
	})

	JustBeforeEach(func() {
		logger = lagertest.NewTestLogger("Cluster test")
		cluster = domain.NewCluster(backends, healthcheckTimeout, logger, fakeArpManager)
	})

	Describe("Monitor", func() {
		var urlGetter *domainfakes.FakeUrlGetter
		var healthyResponse = &http.Response{
			Body:       ioutil.NopCloser(bytes.NewBuffer(nil)),
			StatusCode: http.StatusOK,
		}
		var stopMonitoring chan interface{}

		BeforeEach(func() {
			urlGetter = new(domainfakes.FakeUrlGetter)
			urlGetter := urlGetter
			domain.UrlGetterProvider = func(time.Duration) domain.UrlGetter {
				return urlGetter
			}

			urlGetter.GetReturns(healthyResponse, nil)

			stopMonitoring = make(chan interface{})
		})

		AfterEach(func() {
			domain.UrlGetterProvider = domain.HttpUrlGetterProvider
			close(stopMonitoring)
		})

		It("notices when each backend stays healthy", func(done Done) {
			cluster.Monitor(stopMonitoring)

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

			close(done)
		}, 5)

		It("notices when a healthy backend becomes unhealthy", func() {
			unhealthyResponse := &http.Response{
				Body:       ioutil.NopCloser(bytes.NewBuffer(nil)),
				StatusCode: http.StatusInternalServerError,
			}

			urlGetter.GetStub = func(url string) (*http.Response, error) {
				if url == "backend2" {
					return unhealthyResponse, nil
				} else {
					return healthyResponse, nil
				}
			}

			cluster.Monitor(stopMonitoring)

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

			cluster.Monitor(stopMonitoring)

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
				Body:       ioutil.NopCloser(bytes.NewBuffer(nil)),
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

			cluster.Monitor(stopMonitoring)

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
				fakeArpManager = new(domainfakes.FakeArpManager)
			})

			It("does not clears arp cache after ArpFlushInterval has elapsed", func() {
				cluster.Monitor(stopMonitoring)

				Consistently(fakeArpManager.ClearCacheCallCount, healthcheckTimeout*2).Should(BeZero())
			})
		})

		Context("when a backend is unhealthy", func() {

			BeforeEach(func() {
				fakeArpManager = new(domainfakes.FakeArpManager)
				unhealthyResponse := &http.Response{
					Body:       ioutil.NopCloser(bytes.NewBuffer(nil)),
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

					cluster.Monitor(stopMonitoring)

					Eventually(fakeArpManager.ClearCacheCallCount, 10*time.Second, 500*time.Millisecond).Should(BeNumerically(">=", 1), "Expected arpManager.ClearCache to be called at least once")
					Expect(fakeArpManager.ClearCacheArgsForCall(0)).To(Equal(backend2.AsJSON().Host))
				})
			})

			Context("and the IP is not in the ARP cache", func() {

				BeforeEach(func() {
					fakeArpManager.IsCachedReturns(false)
				})

				It("does not clear arp cache", func() {
					cluster.Monitor(stopMonitoring)

					Consistently(fakeArpManager.ClearCacheCallCount, healthcheckTimeout*2).Should(BeZero())
				})
			})
		})
	})

	Describe("RouteToBackend", func() {
		var clientConn net.Conn

		BeforeEach(func() {
			clientConn = new(domainfakes.FakeConn)
		})

		It("bridges the client connection to the active backend", func() {
			activeBackend := new(domainfakes.FakeBackend)
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

	Describe("EnableTraffic", func() {
		var (
			message string
		)

		BeforeEach(func() {
			message = "some message"
		})

		It("calls EnableTraffic on all the backends", func() {
			cluster.EnableTraffic(message)

			for _, backend := range backendSlice {
				Expect(backend.EnableTrafficCallCount()).To(Equal(1))
			}
		})

		It("records the message", func() {
			cluster.EnableTraffic(message)

			clusterJSON := cluster.AsJSON()

			Expect(clusterJSON.Message).To(Equal(message))
		})

		It("records the current time", func() {
			beforeTime := time.Now()
			cluster.EnableTraffic(message)
			afterTime := time.Now()

			clusterJSON := cluster.AsJSON()

			Expect(clusterJSON.LastUpdated.After(beforeTime)).To(BeTrue())
			Expect(clusterJSON.LastUpdated.Before(afterTime)).To(BeTrue())
		})
	})

	Describe("DisableTraffic", func() {
		var (
			message string
		)

		BeforeEach(func() {
			message = "some message"
		})

		It("calls DisableTraffic on all the backends", func() {
			cluster.DisableTraffic(message)

			for _, backend := range backendSlice {
				Expect(backend.DisableTrafficCallCount()).To(Equal(1))
			}
		})

		It("records the message", func() {
			cluster.DisableTraffic(message)

			clusterJSON := cluster.AsJSON()

			Expect(clusterJSON.Message).To(Equal(message))
		})

		It("records the current time", func() {
			beforeTime := time.Now()
			cluster.DisableTraffic(message)
			afterTime := time.Now()

			clusterJSON := cluster.AsJSON()

			Expect(clusterJSON.LastUpdated.After(beforeTime)).To(BeTrue())
			Expect(clusterJSON.LastUpdated.Before(afterTime)).To(BeTrue())
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
