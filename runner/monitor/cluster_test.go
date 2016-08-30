package monitor_test

import (
	"errors"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"bytes"
	"io/ioutil"

	"github.com/cloudfoundry-incubator/switchboard/domain"
	"github.com/cloudfoundry-incubator/switchboard/domain/domainfakes"
	. "github.com/cloudfoundry-incubator/switchboard/runner/monitor"
	"github.com/cloudfoundry-incubator/switchboard/runner/monitor/monitorfakes"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"
)

const healthcheckTimeout = time.Second

var _ = Describe("Cluster", func() {
	var (
		backends                     *monitorfakes.FakeBackends
		backendSlice                 []*domainfakes.FakeBackend
		logger                       lager.Logger
		cluster                      *Cluster
		fakeArpManager               *monitorfakes.FakeArpManager
		backend1, backend2, backend3 *domainfakes.FakeBackend
	)

	BeforeEach(func() {
		fakeArpManager = nil
		backends = new(monitorfakes.FakeBackends)

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
			c := make(chan domain.Backend, 3)

			for _, b := range backendSlice {
				c <- b
			}
			close(c)

			return c
		}
	})

	JustBeforeEach(func() {
		logger = lagertest.NewTestLogger("Cluster test")
		cluster = NewCluster(backends, healthcheckTimeout, logger, fakeArpManager)
	})

	Describe("Monitor", func() {
		var urlGetter *monitorfakes.FakeUrlGetter
		var healthyResponse = &http.Response{
			Body:       ioutil.NopCloser(bytes.NewBuffer(nil)),
			StatusCode: http.StatusOK,
		}
		var stopMonitoring chan interface{}

		BeforeEach(func() {
			urlGetter = new(monitorfakes.FakeUrlGetter)
			urlGetter := urlGetter
			UrlGetterProvider = func(time.Duration) UrlGetter {
				return urlGetter
			}

			urlGetter.GetReturns(healthyResponse, nil)

			stopMonitoring = make(chan interface{})
		})

		AfterEach(func() {
			UrlGetterProvider = HttpUrlGetterProvider
			close(stopMonitoring)
		})

		It("notices when each backend stays healthy", func(done Done) {
			// backendStates := make(map[domain.Backend]bool)
			// backends.SetStateStub = func(backend domain.Backend, healthy bool) {
			// 	backendStates[backend] = healthy
			// }

			healthyBackends := make(map[domain.Backend]interface{})
			unhealthyBackends := make(map[domain.Backend]interface{})
			backends.SetStateStub = func(backend domain.Backend, healthy bool) {
				if healthy {
					delete(healthyBackends, backend)
					healthyBackends[backend] = struct{}{}
				} else {
					delete(healthyBackends, backend)
					unhealthyBackends[backend] = struct{}{}
				}
			}

			cluster.Monitor(stopMonitoring)

			// cluster.Monitor(stopMonitoring)

			// Expect(len(backendStates)).To(Equal(len(backendSlice)))
			// Expect(backendStates[backend1]).To(BeTrue())
			// Expect(backendStates[backend2]).To(BeTrue())
			// Expect(backendStates[backend3]).To(BeTrue())

			Expect(healthyBackends.Keys()).To(ConsistOf(backendSlice))
			Expect(healthyBackends).To(BeEmpty())

			for i := 0; i < backends.SetStateCallCount(); i++ {
				_, healthy := backends.SetStateArgsForCall(i)
				Expect(healthy).To(BeTrue())
			}

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
				fakeArpManager = new(monitorfakes.FakeArpManager)
			})

			It("does not clears arp cache after ArpFlushInterval has elapsed", func() {
				cluster.Monitor(stopMonitoring)

				Consistently(fakeArpManager.ClearCacheCallCount, healthcheckTimeout*2).Should(BeZero())
			})
		})

		Context("when a backend is unhealthy", func() {

			BeforeEach(func() {
				fakeArpManager = new(monitorfakes.FakeArpManager)
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
