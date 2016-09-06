package monitor_test

import (
	"errors"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"bytes"
	"io/ioutil"

	"sync"

	"github.com/cloudfoundry-incubator/switchboard/domain"
	. "github.com/cloudfoundry-incubator/switchboard/runner/monitor"
	"github.com/cloudfoundry-incubator/switchboard/runner/monitor/monitorfakes"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"
)

const healthcheckTimeout = time.Second

var _ = Describe("Cluster", func() {
	var (
		backends                     []*domain.Backend
		logger                       lager.Logger
		cluster                      *Cluster
		fakeArpManager               *monitorfakes.FakeArpManager
		backend1, backend2, backend3 *domain.Backend
		subscriberA                  chan *domain.Backend
		subscriberB                  chan *domain.Backend
		activeBackendSubscribers     []chan<- *domain.Backend

		m sync.RWMutex
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("Cluster test")
		fakeArpManager = nil

		backend1 = domain.NewBackend(
			"backend-1",
			"10.10.1.2",
			1337,
			1338,
			"healthcheck",
			logger,
		)

		m.Lock()
		backend2 = domain.NewBackend(
			"backend-2",
			"10.10.2.2",
			1337,
			1338,
			"healthcheck",
			logger,
		)
		m.Unlock()

		backend3 = domain.NewBackend(
			"backend-3",
			"10.10.3.2",
			1337,
			1338,
			"healthcheck",
			logger,
		)

		backends = []*domain.Backend{
			backend1,
			backend2,
			backend3,
		}

		subscriberA = make(chan *domain.Backend, 100)
		subscriberB = make(chan *domain.Backend, 100)
		activeBackendSubscribers = []chan<- *domain.Backend{
			subscriberA,
			subscriberB,
		}

		backend1.SetHealthy()
		backend2.SetHealthy()
		backend3.SetHealthy()
	})

	JustBeforeEach(func() {
		cluster = NewCluster(backends, healthcheckTimeout, logger, fakeArpManager, activeBackendSubscribers)
	})

	Describe("Monitor", func() {
		var urlGetter *monitorfakes.FakeUrlGetter
		var healthyResponse = &http.Response{
			Body:       ioutil.NopCloser(bytes.NewBuffer(nil)),
			StatusCode: http.StatusOK,
		}

		BeforeEach(func() {
			urlGetter = new(monitorfakes.FakeUrlGetter)
			UrlGetterProvider = func(time.Duration) UrlGetter {
				return urlGetter
			}

			urlGetter.GetReturns(healthyResponse, nil)
		})

		AfterEach(func() {
			UrlGetterProvider = HttpUrlGetterProvider
		})

		It("notices when each backend stays healthy", func(done Done) {
			backend1.SetUnhealthy()
			backend2.SetUnhealthy()
			backend3.SetUnhealthy()

			Expect(backend1.Healthy()).To(BeFalse())
			Expect(backend2.Healthy()).To(BeFalse())
			Expect(backend3.Healthy()).To(BeFalse())

			cluster.Monitor(nil)

			Eventually(backend1.Healthy).Should(BeTrue())
			Eventually(backend2.Healthy).Should(BeTrue())
			Eventually(backend3.Healthy).Should(BeTrue())

			close(done)
		}, 5)

		It("notices when a healthy backend becomes unhealthy", func() {
			unhealthyResponse := &http.Response{
				Body:       ioutil.NopCloser(bytes.NewBuffer(nil)),
				StatusCode: http.StatusInternalServerError,
			}

			urlGetter.GetStub = func(url string) (*http.Response, error) {
				m.RLock()
				defer m.RUnlock()
				if url == backend2.HealthcheckUrl() {

					return unhealthyResponse, nil
				} else {
					return healthyResponse, nil
				}
			}

			Expect(backend1.Healthy()).To(BeTrue())
			Expect(backend2.Healthy()).To(BeTrue())
			Expect(backend3.Healthy()).To(BeTrue())

			cluster.Monitor(nil)

			Eventually(backend2.Healthy).Should(BeFalse())
			Consistently(backend1.Healthy).Should(BeTrue())
			Consistently(backend3.Healthy).Should(BeTrue())
		})

		It("notices when a healthy backend becomes unresponsive", func() {

			urlGetter.GetStub = func(url string) (*http.Response, error) {
				m.RLock()
				defer m.RUnlock()
				if url == backend2.HealthcheckUrl() {
					return nil, errors.New("some error")
				} else {
					return healthyResponse, nil
				}
			}

			Expect(backend1.Healthy()).To(BeTrue())
			Expect(backend2.Healthy()).To(BeTrue())
			Expect(backend3.Healthy()).To(BeTrue())

			cluster.Monitor(nil)

			Eventually(backend2.Healthy).Should(BeFalse())
			Consistently(backend1.Healthy).Should(BeTrue())
			Consistently(backend3.Healthy).Should(BeTrue())
		})

		It("notices when an unhealthy backend becomes healthy", func() {
			backend2.SetUnhealthy()

			unhealthyResponse := &http.Response{
				Body:       ioutil.NopCloser(bytes.NewBuffer(nil)),
				StatusCode: http.StatusInternalServerError,
			}

			isUnhealthy := true
			urlGetter.GetStub = func(url string) (*http.Response, error) {
				m.RLock()
				defer m.RUnlock()
				if url == backend2.HealthcheckUrl() && isUnhealthy {
					isUnhealthy = false
					return unhealthyResponse, nil
				} else {
					return healthyResponse, nil
				}
			}

			Expect(backend1.Healthy()).To(BeTrue())
			Expect(backend2.Healthy()).To(BeFalse())
			Expect(backend3.Healthy()).To(BeTrue())

			cluster.Monitor(nil)

			Eventually(backend2.Healthy).Should(BeTrue())
			Consistently(backend1.Healthy).Should(BeTrue())
			Consistently(backend3.Healthy).Should(BeTrue())
		})

		Context("when the active backend changes", func() {
			It("publishes the new backend", func() {
				cluster.Monitor(nil)
				var firstActive *domain.Backend
				Eventually(subscriberA).Should(Receive(&firstActive))
				Eventually(subscriberB).Should(Receive(&firstActive))

				urlGetter.GetStub = func(url string) (*http.Response, error) {
					m.RLock()
					defer m.RUnlock()
					if url == firstActive.HealthcheckUrl() {
						return nil, errors.New("some error")
					} else {
						return healthyResponse, nil
					}
				}

				Eventually(subscriberA).Should(Receive(Not(Equal(firstActive))))
				Eventually(subscriberB).Should(Receive(Not(Equal(firstActive))))
			})
		})

		Context("when a backend is healthy", func() {
			BeforeEach(func() {
				fakeArpManager = new(monitorfakes.FakeArpManager)
			})

			It("does not clears arp cache after ArpFlushInterval has elapsed", func() {
				cluster.Monitor(nil)

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
					m.RLock()
					defer m.RUnlock()
					if url == backend2.HealthcheckUrl() {
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

					cluster.Monitor(nil)

					Eventually(fakeArpManager.ClearCacheCallCount, 10*time.Second, 500*time.Millisecond).Should(BeNumerically(">=", 1), "Expected arpManager.ClearCache to be called at least once")
					Expect(fakeArpManager.ClearCacheArgsForCall(0)).To(Equal(backend2.AsJSON().Host))
				})
			})

			Context("and the IP is not in the ARP cache", func() {

				BeforeEach(func() {
					fakeArpManager.IsCachedReturns(false)
				})

				It("does not clear arp cache", func() {
					cluster.Monitor(nil)

					Consistently(fakeArpManager.ClearCacheCallCount, healthcheckTimeout*2).Should(BeZero())
				})
			})
		})
	})

	Describe("ChooseActiveBackend", func() {
		var (
			statuses                     map[*domain.Backend]*BackendStatus
			backend1, backend2, backend3 *domain.Backend
		)

		BeforeEach(func() {
			statuses = make(map[*domain.Backend]*BackendStatus)
			backend1 = domain.NewBackend(
				"backend-1",
				"10.10.1.2",
				1337,
				1338,
				"healthcheck",
				logger,
			)

			backend2 = domain.NewBackend(
				"backend-2",
				"10.10.2.2",
				1337,
				1338,
				"healthcheck",
				logger,
			)
			backend3 = domain.NewBackend(
				"backend-3",
				"10.10.3.2",
				1337,
				1338,
				"healthcheck",
				logger,
			)
		})

		Context("When there are no backends", func() {
			It("returns nil", func() {
				Expect(ChooseActiveBackend(statuses)).To(BeNil())
			})
		})
		Context("If none of the backends are healthy", func() {
			It("returns nil", func() {
				statuses[backend1] = &BackendStatus{
					Healthy: false,
					Index:   0,
				}

				statuses[backend2] = &BackendStatus{
					Healthy: false,
					Index:   1,
				}

				statuses[backend3] = &BackendStatus{
					Healthy: false,
					Index:   2,
				}

				Expect(ChooseActiveBackend(statuses)).To(BeNil())
			})
		})

		Context("If only one of the backends is healthy", func() {
			It("chooses the only healthy one", func() {
				statuses[backend1] = &BackendStatus{
					Healthy: false,
					Index:   0,
				}

				statuses[backend2] = &BackendStatus{
					Healthy: false,
					Index:   1,
				}

				statuses[backend3] = &BackendStatus{
					Healthy: true,
					Index:   2,
				}

				Expect(ChooseActiveBackend(statuses)).To(Equal(backend3))
			})
		})

		Context("If multiple backends are healthy", func() {
			It("chooses the healthy one with the lowest index", func() {
				statuses[backend2] = &BackendStatus{
					Healthy: true,
					Index:   2,
				}

				statuses[backend3] = &BackendStatus{
					Healthy: true,
					Index:   1,
				}

				statuses[backend1] = &BackendStatus{
					Healthy: false,
					Index:   0,
				}

				Expect(ChooseActiveBackend(statuses)).To(Equal(backend3))
			})
		})

	})
})
