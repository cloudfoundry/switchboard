package api_test

import (
	"encoding/json"
	"net/http"
	"time"

	"code.cloudfoundry.org/lager/lagertest"
	"github.com/cloudfoundry-incubator/switchboard/api"
	"github.com/cloudfoundry-incubator/switchboard/api/apifakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("ClusterEndpoint", func() {
	var (
		fakeCluster *apifakes.FakeClusterManager
		testLogger  *lagertest.TestLogger

		handler http.HandlerFunc

		server *ghttp.Server
	)

	BeforeEach(func() {
		fakeCluster = new(apifakes.FakeClusterManager)

		testLogger = lagertest.NewTestLogger("Switchboard API test")

		handler = api.ClusterEndpoint(fakeCluster, testLogger)

		server = ghttp.NewServer()
		server.AppendHandlers(handler)
	})

	Describe("GET", func() {
		It("returns 200", func() {
			req, err := http.NewRequest("GET", server.URL(), nil)
			Expect(err).NotTo(HaveOccurred())

			client := &http.Client{}
			resp, err := client.Do(req)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		It("contains expected fields", func() {
			updateTime := time.Now()

			expectedClusterJSON := api.ClusterJSON{
				TrafficEnabled: true,
				LastUpdated:    updateTime,
			}
			fakeCluster.AsJSONReturns(expectedClusterJSON)

			req, err := http.NewRequest("GET", server.URL(), nil)
			Expect(err).NotTo(HaveOccurred())

			client := &http.Client{}
			resp, err := client.Do(req)
			Expect(err).NotTo(HaveOccurred())

			var returnedCluster api.ClusterJSON
			decoder := json.NewDecoder(resp.Body)
			err = decoder.Decode(&returnedCluster)
			Expect(err).NotTo(HaveOccurred())

			Expect(returnedCluster.TrafficEnabled).To(BeTrue())
			Expect(returnedCluster.LastUpdated.Second()).To(Equal(updateTime.Second()))
		})
	})

	Describe("PATCH", func() {
		var (
			patchURL string
		)

		BeforeEach(func() {
			patchURL = server.URL() + "?trafficEnabled=true"
		})

		It("returns 200", func() {
			req, err := http.NewRequest("PATCH", patchURL, nil)
			Expect(err).NotTo(HaveOccurred())

			client := &http.Client{}
			resp, err := client.Do(req)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		It("contains expected fields", func() {
			expectedClusterJSON := api.ClusterJSON{
				TrafficEnabled: true,
				Message:        "some reason",
			}
			fakeCluster.AsJSONReturns(expectedClusterJSON)

			req, err := http.NewRequest("PATCH", patchURL, nil)
			Expect(err).NotTo(HaveOccurred())

			client := &http.Client{}
			resp, err := client.Do(req)
			Expect(err).NotTo(HaveOccurred())

			var returnedCluster api.ClusterJSON
			decoder := json.NewDecoder(resp.Body)
			err = decoder.Decode(&returnedCluster)
			Expect(err).NotTo(HaveOccurred())

			Expect(returnedCluster.TrafficEnabled).To(BeTrue())
			Expect(returnedCluster.Message).To(Equal("some reason"))
		})

		Context("when traffic is enabled", func() {
			BeforeEach(func() {
				patchURL = server.URL() + "?trafficEnabled=true"
			})

			It("invokes cluster.EnableTraffic", func() {
				req, err := http.NewRequest("PATCH", patchURL, nil)
				Expect(err).NotTo(HaveOccurred())

				client := &http.Client{}
				_, err = client.Do(req)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeCluster.EnableTrafficCallCount()).To(Equal(1))
			})

			It("records a message when provided", func() {
				patchURL = server.URL() + "?trafficEnabled=true&message=some%20message"
				req, err := http.NewRequest("PATCH", patchURL, nil)
				Expect(err).NotTo(HaveOccurred())

				client := &http.Client{}
				_, err = client.Do(req)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeCluster.EnableTrafficCallCount()).To(Equal(1))
				Expect(fakeCluster.EnableTrafficArgsForCall(0)).To(Equal("some message"))
			})

			It("does not require a message", func() {
				req, err := http.NewRequest("PATCH", patchURL, nil)
				Expect(err).NotTo(HaveOccurred())

				client := &http.Client{}
				_, err = client.Do(req)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeCluster.EnableTrafficCallCount()).To(Equal(1))
				Expect(fakeCluster.EnableTrafficArgsForCall(0)).To(BeEmpty())
			})

			Context("when traffic is disabled", func() {
				BeforeEach(func() {
					patchURL = server.URL() + "?trafficEnabled=false&message=some%20message"
				})

				It("invokes cluster.DisableTraffic", func() {
					req, err := http.NewRequest("PATCH", patchURL, nil)
					Expect(err).NotTo(HaveOccurred())

					client := &http.Client{}
					resp, err := client.Do(req)
					Expect(err).NotTo(HaveOccurred())

					Expect(resp.StatusCode).To(Equal(http.StatusOK))

					Expect(fakeCluster.DisableTrafficCallCount()).To(Equal(1))
					Expect(fakeCluster.DisableTrafficArgsForCall(0)).To(Equal("some message"))
				})

				It("requires a message", func() {
					patchURL = server.URL()
					req, err := http.NewRequest("PATCH", patchURL, nil)
					Expect(err).NotTo(HaveOccurred())

					client := &http.Client{}
					resp, err := client.Do(req)
					Expect(err).NotTo(HaveOccurred())

					Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
				})
			})

			Context("when the URL is missing trafficEnabled", func() {
				It("returns 400 - Bad request", func() {
					url := server.URL()
					req, err := http.NewRequest("PATCH", url, nil)
					Expect(err).NotTo(HaveOccurred())

					client := &http.Client{}
					resp, err := client.Do(req)
					Expect(err).NotTo(HaveOccurred())

					Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
				})
			})

			Context("when the URL has an unparsable value for trafficEnabled", func() {
				It("returns 400 - Bad request", func() {
					url := server.URL() + "?trafficEnabled=unparsable"
					req, err := http.NewRequest("PATCH", url, nil)
					Expect(err).NotTo(HaveOccurred())

					client := &http.Client{}
					resp, err := client.Do(req)
					Expect(err).NotTo(HaveOccurred())

					Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
				})
			})
		})

		Describe("POST", func() {
			It("returns http 405 - Method not allowed", func() {
				req, err := http.NewRequest("POST", server.URL(), nil)
				Expect(err).NotTo(HaveOccurred())

				client := &http.Client{}
				resp, err := client.Do(req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))
			})
		})

		Describe("PUT", func() {
			It("returns http 405 - Method not allowed", func() {
				req, err := http.NewRequest("PUT", server.URL(), nil)
				Expect(err).NotTo(HaveOccurred())

				client := &http.Client{}
				resp, err := client.Do(req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))
			})
		})

		Describe("DELETE", func() {
			It("returns http 405 - Method not allowed", func() {
				req, err := http.NewRequest("DELETE", server.URL(), nil)
				Expect(err).NotTo(HaveOccurred())

				client := &http.Client{}
				resp, err := client.Do(req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))
			})
		})
	})
})
