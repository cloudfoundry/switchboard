package api_test

import (
	"encoding/json"
	"net/http"

	"github.com/cloudfoundry-incubator/switchboard/api"
	"github.com/cloudfoundry-incubator/switchboard/domain"
	domainfakes "github.com/cloudfoundry-incubator/switchboard/domain/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/pivotal-golang/lager/lagertest"
)

var _ = Describe("Cluster", func() {
	var (
		fakeCluster *domainfakes.FakeCluster
		testLogger  *lagertest.TestLogger

		handler http.HandlerFunc

		server *ghttp.Server
	)

	BeforeEach(func() {
		fakeCluster = &domainfakes.FakeCluster{}

		testLogger = lagertest.NewTestLogger("Switchboard API test")

		handler = api.Cluster(fakeCluster, testLogger)

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
			expectedClusterJSON := domain.ClusterJSON{
				TrafficEnabled:      true,
				CurrentBackendIndex: 2,
			}
			fakeCluster.AsJSONReturns(expectedClusterJSON)

			req, err := http.NewRequest("GET", server.URL(), nil)
			Expect(err).NotTo(HaveOccurred())

			client := &http.Client{}
			resp, err := client.Do(req)
			Expect(err).NotTo(HaveOccurred())

			var returnedCluster domain.ClusterJSON
			decoder := json.NewDecoder(resp.Body)
			err = decoder.Decode(&returnedCluster)
			Expect(err).NotTo(HaveOccurred())

			Expect(returnedCluster.TrafficEnabled).To(BeTrue())
			Expect(returnedCluster.CurrentBackendIndex).To(BeNumerically("==", 2))
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
			expectedClusterJSON := domain.ClusterJSON{
				TrafficEnabled:      true,
				CurrentBackendIndex: 2,
				Message:             "some reason",
			}
			fakeCluster.AsJSONReturns(expectedClusterJSON)

			req, err := http.NewRequest("PATCH", patchURL, nil)
			Expect(err).NotTo(HaveOccurred())

			client := &http.Client{}
			resp, err := client.Do(req)
			Expect(err).NotTo(HaveOccurred())

			var returnedCluster domain.ClusterJSON
			decoder := json.NewDecoder(resp.Body)
			err = decoder.Decode(&returnedCluster)
			Expect(err).NotTo(HaveOccurred())

			Expect(returnedCluster.TrafficEnabled).To(BeTrue())
			Expect(returnedCluster.CurrentBackendIndex).To(BeNumerically("==", 2))
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
