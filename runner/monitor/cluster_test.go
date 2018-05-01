package monitor_test

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"bytes"
	"io/ioutil"

	"sync"

	"strings"

	"code.cloudfoundry.org/lager/lagertest"
	"github.com/cloudfoundry-incubator/switchboard/domain"
	. "github.com/cloudfoundry-incubator/switchboard/runner/monitor"
	"github.com/cloudfoundry-incubator/switchboard/runner/monitor/monitorfakes"
)

const healthcheckTimeout = 500 * time.Millisecond

var _ = Describe("Cluster", func() {
	var (
		backends                     []*domain.Backend
		logger                       *lagertest.TestLogger
		cluster                      *Cluster
		backend1, backend2, backend3 *domain.Backend
		subscriberA                  chan *domain.Backend
		subscriberB                  chan *domain.Backend
		activeBackendSubscribers     []chan<- *domain.Backend
		notFoundResponse             *http.Response

		m sync.RWMutex
	)

	BeforeEach(func() {
		cluster = nil

		logger = lagertest.NewTestLogger("Cluster test")

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

		notFoundResponse = &http.Response{
			Body:       ioutil.NopCloser(bytes.NewBuffer(nil)),
			StatusCode: http.StatusNotFound,
		}
	})

	JustBeforeEach(func() {
		cluster = NewCluster(
			backends,
			healthcheckTimeout,
			logger,
			activeBackendSubscribers,
		)
	})

	Describe("Monitor", func() {
		var (
			urlGetter       *monitorfakes.FakeUrlGetter
			healthyResponse *http.Response

			stopMonitoringChan chan interface{}
		)

		BeforeEach(func() {
			stopMonitoringChan = make(chan interface{})

			urlGetter = new(monitorfakes.FakeUrlGetter)
			UrlGetterProvider = func(time.Duration) UrlGetter {
				return urlGetter
			}

			healthyResponse = &http.Response{
				Body:       ioutil.NopCloser(bytes.NewBuffer(nil)),
				StatusCode: http.StatusOK,
			}

			urlGetter.GetStub = func(url string) (*http.Response, error) {
				m.RLock()
				defer m.RUnlock()
				if strings.HasSuffix(url, "api/v1/status") {
					return notFoundResponse, nil
				}
				return healthyResponse, nil
			}
		})

		AfterEach(func() {
			UrlGetterProvider = HttpUrlGetterProvider

			close(stopMonitoringChan)
		})

		It("notices when each backend stays healthy", func(done Done) {
			backend1.SetUnhealthy()
			backend2.SetUnhealthy()
			backend3.SetUnhealthy()

			Expect(backend1.Healthy()).To(BeFalse())
			Expect(backend2.Healthy()).To(BeFalse())
			Expect(backend3.Healthy()).To(BeFalse())

			cluster.Monitor(stopMonitoringChan)

			Eventually(backend1.Healthy).Should(BeTrue())
			Eventually(backend2.Healthy).Should(BeTrue())
			Eventually(backend3.Healthy).Should(BeTrue())

			close(done)
		}, 5)

		It("notices when a healthy backend becomes unhealthy", func(done Done) {
			unhealthyResponse := &http.Response{
				Body:       ioutil.NopCloser(bytes.NewBuffer(nil)),
				StatusCode: http.StatusInternalServerError,
			}

			urlGetter.GetStub = func(url string) (*http.Response, error) {
				m.RLock()
				defer m.RUnlock()
				if strings.HasSuffix(url, "api/v1/status") {
					return notFoundResponse, nil
				}
				if url == backend2.HealthcheckUrl() {
					return unhealthyResponse, nil
				} else {
					return healthyResponse, nil
				}
			}

			Expect(backend1.Healthy()).To(BeTrue())
			Expect(backend2.Healthy()).To(BeTrue())
			Expect(backend3.Healthy()).To(BeTrue())

			cluster.Monitor(stopMonitoringChan)

			Eventually(backend2.Healthy).Should(BeFalse())
			Consistently(backend1.Healthy).Should(BeTrue())
			Consistently(backend3.Healthy).Should(BeTrue())
			close(done)
		}, 5)

		It("notices when a healthy backend becomes unresponsive", func() {
			urlGetter.GetStub = func(url string) (*http.Response, error) {
				m.RLock()
				defer m.RUnlock()
				if strings.HasSuffix(url, "api/v1/status") {
					return notFoundResponse, nil
				}
				if url == backend2.HealthcheckUrl() {
					return nil, errors.New("some error")
				} else {
					return healthyResponse, nil
				}
			}

			Expect(backend1.Healthy()).To(BeTrue())
			Expect(backend2.Healthy()).To(BeTrue())
			Expect(backend3.Healthy()).To(BeTrue())

			cluster.Monitor(stopMonitoringChan)

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
				if strings.HasSuffix(url, "api/v1/status") {
					return notFoundResponse, nil
				}
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

			cluster.Monitor(stopMonitoringChan)

			Eventually(backend2.Healthy).Should(BeTrue())
			Consistently(backend1.Healthy).Should(BeTrue())
			Consistently(backend3.Healthy).Should(BeTrue())
		})

		Context("when the active backend changes", func() {
			It("publishes the new backend", func() {
				cluster.Monitor(stopMonitoringChan)
				var firstActive *domain.Backend
				Eventually(subscriberA).Should(Receive(&firstActive))
				Eventually(subscriberB).Should(Receive(&firstActive))

				urlGetter.GetStub = func(url string) (*http.Response, error) {
					m.RLock()
					defer m.RUnlock()

					if strings.HasSuffix(url, "api/v1/status") {
						return notFoundResponse, nil
					}

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
	})

	Describe("QueryBackendHealth", func() {
		var (
			urlGetter     *monitorfakes.FakeUrlGetter
			backend       *domain.Backend
			backendStatus *BackendStatus

			backendStatusPort uint
			backendHost       string

			v0Err        error
			v0StatusCode int
			v0Response   *http.Response

			v1Err        error
			v1StatusCode int
			v1Response   *http.Response
		)

		BeforeEach(func() {
			v0Err = nil
			v1Err = nil

			v0StatusCode = http.StatusOK
			v1StatusCode = http.StatusOK

			urlGetter = new(monitorfakes.FakeUrlGetter)
			UrlGetterProvider = func(time.Duration) UrlGetter {
				return urlGetter
			}

			backendStatusPort = 9292
			backendHost = "192.0.2.10"

			backend = domain.NewBackend(
				"backend-0",
				backendHost,
				3306,
				backendStatusPort,
				"",
				logger,
			)
		})

		JustBeforeEach(func() {
			v0Response = &http.Response{
				Body:       ioutil.NopCloser(bytes.NewBuffer(nil)),
				StatusCode: v0StatusCode,
			}

			v1Response = &http.Response{
				Body:       ioutil.NopCloser(strings.NewReader(`{"healthy": true, "wsrep_local_index": 0}`)),
				StatusCode: v1StatusCode,
			}

			urlGetter.GetStub = func(url string) (*http.Response, error) {
				m.RLock()
				defer m.RUnlock()

				if strings.Contains(url, "api/v1/") {
					return v1Response, v1Err
				}
				return v0Response, v0Err
			}

			backendStatus = &BackendStatus{
				Index:    2,
				Counters: cluster.SetupCounters(),
				Healthy:  false,
			}
		})

		AfterEach(func() {
			UrlGetterProvider = HttpUrlGetterProvider
		})

		It("changes the backend health and index", func() {
			Expect(backendStatus.Healthy).To(BeFalse())
			Expect(backendStatus.Index).To(Equal(2))

			cluster.QueryBackendHealth(backend, backendStatus, urlGetter)
			Expect(urlGetter.GetCallCount()).To(Equal(1))

			expectedURL := fmt.Sprintf(
				"http://%s:%d/api/v1/status",
				backendHost,
				backendStatusPort,
			)
			Expect(urlGetter.GetArgsForCall(0)).To(Equal(expectedURL))

			Expect(backendStatus.Healthy).To(BeTrue())
			Expect(backendStatus.Index).To(Equal(0))
		})

		Context("when GETting the v1 API returns an error", func() {
			BeforeEach(func() {
				v1Err = errors.New("v1 api not available")
			})

			It("marks the backend as unhealthy", func() {
				backend.SetHealthy()

				cluster.QueryBackendHealth(backend, backendStatus, urlGetter)
				Expect(urlGetter.GetCallCount()).To(Equal(1))

				Expect(backendStatus.Healthy).To(BeFalse())
			})
		})

		Context("when GETting the v1 API returns a 404", func() {
			BeforeEach(func() {
				v1StatusCode = http.StatusNotFound
			})

			It("uses the previous API to set the health to true, ignoring the index", func() {
				Expect(backendStatus.Healthy).To(BeFalse())
				Expect(backendStatus.Index).To(Equal(2))

				cluster.QueryBackendHealth(backend, backendStatus, urlGetter)
				Expect(urlGetter.GetCallCount()).To(Equal(2))

				Expect(backendStatus.Healthy).To(BeTrue())
				Expect(backendStatus.Index).To(Equal(2)) // unchanged
			})

			Context("when GETting the v0 API returns an error", func() {
				BeforeEach(func() {
					v0Err = errors.New("v0 api not available")
				})

				It("uses the previous API to set the health to false, ignoring the index", func() {
					Expect(backendStatus.Healthy).To(BeFalse())
					Expect(backendStatus.Index).To(Equal(2))

					cluster.QueryBackendHealth(backend, backendStatus, urlGetter)
					Expect(urlGetter.GetCallCount()).To(Equal(2))

					Expect(backendStatus.Healthy).To(BeFalse())
					Expect(backendStatus.Index).To(Equal(2)) // unchanged
				})

				Context("when GETting the v0 API returns a bad status code", func() {
					BeforeEach(func() {
						v0StatusCode = http.StatusTeapot
					})

					It("uses the previous API to set the health to false, ignoring the index", func() {
						Expect(backendStatus.Healthy).To(BeFalse())
						Expect(backendStatus.Index).To(Equal(2))

						cluster.QueryBackendHealth(backend, backendStatus, urlGetter)
						Expect(urlGetter.GetCallCount()).To(Equal(2))

						Expect(backendStatus.Healthy).To(BeFalse())
						Expect(backendStatus.Index).To(Equal(2)) // unchanged
					})
				})
			})
		})

		Context("when GETting the v1 API returns a bad status code", func() {
			BeforeEach(func() {
				v1StatusCode = http.StatusTeapot
			})

			It("marks the backend as unhealthy", func() {
				backend.SetHealthy()

				cluster.QueryBackendHealth(backend, backendStatus, urlGetter)
				Expect(urlGetter.GetCallCount()).To(Equal(1))

				Expect(backendStatus.Healthy).To(BeFalse())
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
