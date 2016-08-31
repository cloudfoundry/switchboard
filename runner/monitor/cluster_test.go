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
	. "github.com/tjarratt/gcounterfeiter"
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
		})

		It("notices when each backend stays healthy", func(done Done) {
			cluster.Monitor(stopMonitoring)
			time.Sleep(time.Second)
			stopMonitoring <- struct{}{}
			close(stopMonitoring)

			for _, b := range backendSlice {
				Expect(backends).To(HaveReceived("SetState").With(b).AndWith(true))
				Expect(backends).ToNot(HaveReceived("SetState").With(b).AndWith(false))
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

			time.Sleep(time.Second)
			stopMonitoring <- struct{}{}
			close(stopMonitoring)

			Expect(backends).To(HaveReceived("SetState").With(backend1).AndWith(true))
			Expect(backends).ToNot(HaveReceived("SetState").With(backend1).AndWith(false))

			Expect(backends).To(HaveReceived("SetState").With(backend2).AndWith(false))
			Expect(backends).ToNot(HaveReceived("SetState").With(backend2).AndWith(true))

			Expect(backends).To(HaveReceived("SetState").With(backend3).AndWith(true))
			Expect(backends).ToNot(HaveReceived("SetState").With(backend3).AndWith(false))
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
			time.Sleep(time.Second)
			stopMonitoring <- struct{}{}
			close(stopMonitoring)

			Expect(backends).To(HaveReceived("SetState").With(backend1).AndWith(true))
			Expect(backends).ToNot(HaveReceived("SetState").With(backend1).AndWith(false))

			Expect(backends).To(HaveReceived("SetState").With(backend2).AndWith(false))
			Expect(backends).ToNot(HaveReceived("SetState").With(backend2).AndWith(true))

			Expect(backends).To(HaveReceived("SetState").With(backend3).AndWith(true))
			Expect(backends).ToNot(HaveReceived("SetState").With(backend3).AndWith(false))
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
			time.Sleep(time.Second)
			stopMonitoring <- struct{}{}
			close(stopMonitoring)

			Expect(backends).To(HaveReceived("SetState").With(backend1).AndWith(true))
			Expect(backends).ToNot(HaveReceived("SetState").With(backend1).AndWith(false))

			Expect(backends).To(HaveReceived("SetState").With(backend2).AndWith(true))
			Expect(backends).To(HaveReceived("SetState").With(backend2).AndWith(false))

			Expect(backends).To(HaveReceived("SetState").With(backend3).AndWith(true))
			Expect(backends).ToNot(HaveReceived("SetState").With(backend3).AndWith(false))
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
