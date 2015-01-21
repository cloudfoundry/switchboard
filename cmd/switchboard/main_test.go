package main_test

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/switchboard/cmd/switchboard/dummies"
	"github.com/pivotal-cf-experimental/switchboard/config"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
	"github.com/tedsuo/ifrit/grouper"
)

type Response struct {
	BackendPort  uint
	BackendIndex uint
	Message      string
}

func sendData(conn net.Conn, data string) (Response, error) {
	conn.Write([]byte(data))
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return Response{}, err
	} else {
		response := Response{}
		err := json.Unmarshal(buffer[:n], &response)
		if err != nil {
			return Response{}, err
		}
		return response, nil
	}
}

var _ = Describe("Switchboard", func() {
	var process ifrit.Process
	var initialActiveBackend, initialInactiveBackend config.Backend
	var healthcheckRunners []*dummies.HealthcheckRunner

	BeforeEach(func() {
		healthcheckRunners = []*dummies.HealthcheckRunner{
			dummies.NewHealthcheckRunner(backends[0]),
			dummies.NewHealthcheckRunner(backends[1]),
		}

		switchboardRunner := ginkgomon.New(ginkgomon.Config{
			Command: exec.Command(
				switchboardBinPath,
				fmt.Sprintf("-configFile=%s", proxyConfigFile),
				fmt.Sprintf("-pidFile=%s", pidFile),
			),
			Name:       fmt.Sprintf("switchboard"),
			StartCheck: "started",
		})

		group := grouper.NewParallel(os.Kill, grouper.Members{
			grouper.Member{"backend-0", dummies.NewBackendRunner(0, backends[0])},
			grouper.Member{"backend-1", dummies.NewBackendRunner(1, backends[1])},
			grouper.Member{"healthcheck-0", healthcheckRunners[0]},
			grouper.Member{"healthcheck-1", healthcheckRunners[1]},
			grouper.Member{"switchboard", switchboardRunner},
		})
		process = ifrit.Invoke(group)

		var err error
		var conn net.Conn
		Eventually(func() error {
			conn, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
			return err
		}).ShouldNot(HaveOccurred())

		response, err := sendData(conn, "detect active")
		Expect(err).NotTo(HaveOccurred())

		initialActiveBackend = backends[response.BackendIndex]
		initialInactiveBackend = backends[(response.BackendIndex+1)%2]
	})

	AfterEach(func() {
		ginkgomon.Kill(process)
	})

	Describe("api", func() {
		Describe("/servers/", func() {
			var url string

			BeforeEach(func() {
				url = fmt.Sprintf("http://localhost:%d/v0/backends", switchboardAPIPort)
			})

			It("prompts for Basic Auth creds when they aren't provided", func() {
				resp, err := http.Get(url)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
				Expect(resp.Header.Get("WWW-Authenticate")).To(Equal(`Basic realm="Authorization Required"`))
			})

			It("does not accept bad Basic Auth creds", func() {
				req, err := http.NewRequest("GET", url, nil)
				req.SetBasicAuth("bad_username", "bad_password")
				client := &http.Client{}
				resp, err := client.Do(req)

				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			})

			It("returns valid JSON with valid Basic Auth", func() {
				req, err := http.NewRequest("GET", url, nil)
				req.SetBasicAuth("username", "password")
				client := &http.Client{}
				resp, err := client.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				returnedBackends := []map[string]interface{}{}

				decoder := json.NewDecoder(resp.Body)
				err = decoder.Decode(&returnedBackends)
				Expect(err).NotTo(HaveOccurred())

				Expect(len(returnedBackends)).To(Equal(2))

				Expect(returnedBackends[0]["host"]).To(Equal("localhost"))
				Expect(returnedBackends[0]["healthy"]).To(BeTrue())

				Expect(returnedBackends[1]["host"]).To(Equal("localhost"))
				Expect(returnedBackends[1]["healthy"]).To(BeTrue())

				if returnedBackends[0]["active"].(bool) {
					Expect(returnedBackends[0]["currentSessionCount"]).To(BeNumerically("==", 1))
					Expect(returnedBackends[1]["currentSessionCount"]).To(BeNumerically("==", 0))
					Expect(returnedBackends[1]["active"]).To(BeFalse())
				} else {
					Expect(returnedBackends[1]["currentSessionCount"]).To(BeNumerically("==", 1))
					Expect(returnedBackends[0]["currentSessionCount"]).To(BeNumerically("==", 0))
					Expect(returnedBackends[0]["active"]).To(BeFalse())
				}

				switch returnedBackends[0]["name"] {
				case backends[0].BackendName:
					Expect(returnedBackends[0]["port"]).To(BeNumerically("==", backends[0].BackendPort))
					Expect(returnedBackends[1]["port"]).To(BeNumerically("==", backends[1].BackendPort))
					Expect(returnedBackends[1]["name"]).To(Equal(backends[1].BackendName))
				case backends[1].BackendName: // order reversed in response
					Expect(returnedBackends[1]["port"]).To(BeNumerically("==", backends[0].BackendPort))
					Expect(returnedBackends[0]["port"]).To(BeNumerically("==", backends[1].BackendPort))
					Expect(returnedBackends[0]["name"]).To(Equal(backends[1].BackendName))
				default:
					Fail(fmt.Sprintf("Invalid backend name: %s", returnedBackends[0]["name"]))
				}
			})
		})
	})

	Describe("proxy", func() {
		Context("when there are multiple concurrent clients", func() {
			var conn1, conn2, conn3 net.Conn
			var data1, data2, data3 Response

			It("proxies all the connections to the backend", func() {
				done1 := make(chan interface{})
				go func() {
					defer GinkgoRecover()
					defer close(done1)

					var err error
					Eventually(func() error {
						conn1, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
						return err
					}).ShouldNot(HaveOccurred())

					data1, err = sendData(conn1, "test1")
					Expect(err).ToNot(HaveOccurred())
				}()

				done2 := make(chan interface{})
				go func() {
					defer GinkgoRecover()
					defer close(done2)

					var err error
					Eventually(func() error {
						conn2, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
						return err
					}).ShouldNot(HaveOccurred())

					data2, err = sendData(conn2, "test2")
					Expect(err).ToNot(HaveOccurred())
				}()

				done3 := make(chan interface{})
				go func() {
					defer GinkgoRecover()
					defer close(done3)

					var err error
					Eventually(func() error {
						conn3, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
						return err
					}).ShouldNot(HaveOccurred())

					data3, err = sendData(conn3, "test3")
					Expect(err).ToNot(HaveOccurred())
				}()

				<-done1
				<-done2
				<-done3

				Expect(data1.Message).Should(Equal("test1"))
				Expect(data2.Message).Should(Equal("test2"))
				Expect(data3.Message).Should(Equal("test3"))
			})
		})

		Context("when other clients disconnect", func() {
			var conn net.Conn
			var connToDisconnect net.Conn

			It("maintains a long-lived connection when other clients disconnect", func() {
				Eventually(func() error {
					var err error
					conn, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
					return err
				}).ShouldNot(HaveOccurred())

				Eventually(func() error {
					var err error
					connToDisconnect, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
					return err
				}).ShouldNot(HaveOccurred())

				dataBeforeDisconnect, err := sendData(conn, "data before disconnect")
				Expect(err).ToNot(HaveOccurred())
				Expect(dataBeforeDisconnect.Message).Should(Equal("data before disconnect"))

				connToDisconnect.Close()

				dataAfterDisconnect, err := sendData(conn, "data after disconnect")
				Expect(err).ToNot(HaveOccurred())
				Expect(dataAfterDisconnect.Message).Should(Equal("data after disconnect"))
			})
		})

		Context("when the healthcheck succeeds", func() {
			It("checks health again after the specified interval", func() {
				var client net.Conn
				Eventually(func() error {
					var err error
					client, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
					return err
				}).ShouldNot(HaveOccurred())

				data, err := sendData(client, "data around first healthcheck")
				Expect(err).NotTo(HaveOccurred())
				Expect(data.Message).To(Equal("data around first healthcheck"))

				Consistently(func() error {
					_, err = sendData(client, "data around subsequent healthcheck")
					return err
				}, 3*time.Second).ShouldNot(HaveOccurred())
			})
		})

		Context("when the cluster is down", func() {
			Context("when the healthcheck reports a 503", func() {
				It("disconnects client connections", func() {
					var conn net.Conn
					Eventually(func() error {
						var err error
						conn, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
						return err
					}).ShouldNot(HaveOccurred())

					dataWhileHealthy, err := sendData(conn, "data while healthy")
					Expect(err).ToNot(HaveOccurred())
					Expect(dataWhileHealthy.Message).To(Equal("data while healthy"))

					if initialActiveBackend == backends[0] {
						healthcheckRunners[0].SetStatusCode(http.StatusServiceUnavailable)
					} else {
						healthcheckRunners[1].SetStatusCode(http.StatusServiceUnavailable)
					}

					Eventually(func() error {
						_, err := sendData(conn, "data when unhealthy")
						return err
					}, 2*time.Second).Should(HaveOccurred())
				})
			})

			Context("when a backend goes down", func() {
				var conn net.Conn
				var data Response

				BeforeEach(func() {
					Eventually(func() (err error) {
						conn, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
						return err
					}).ShouldNot(HaveOccurred())

					data, err := sendData(conn, "data before hang")
					Expect(err).ShouldNot(HaveOccurred())
					Expect(data.Message).Should(Equal("data before hang"))

					if initialActiveBackend == backends[0] {
						healthcheckRunners[0].SetHang(true)
					} else {
						healthcheckRunners[1].SetHang(true)
					}
				})

				It("disconnects existing client connections", func(done Done) {
					defer close(done)

					Eventually(func() error {
						_, err := sendData(conn, "data after hang")
						return err
					}, proxyConfig.HealthcheckTimeout()*10).Should(HaveOccurred())
				}, 5)

				It("proxies new connections to another backend", func(done Done) {
					defer close(done)

					time.Sleep(3 * proxyConfig.HealthcheckTimeout()) // wait for failover

					var err error
					Eventually(func() error {
						conn, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
						return err
					}).ShouldNot(HaveOccurred())

					data, err = sendData(conn, "test")
					Expect(err).ToNot(HaveOccurred())
					Expect(data.Message).To(Equal("test"))
					Expect(data.BackendPort).To(Equal(initialInactiveBackend.BackendPort))
				}, 5)
			})

			Context("when all backends are down", func() {
				BeforeEach(func() {
					for _, hr := range healthcheckRunners {
						hr.SetHang(true)
					}
				})

				It("rejects any new connections that are attempted", func(done Done) {
					defer close(done)

					time.Sleep(3 * proxyConfig.HealthcheckTimeout()) // wait for failover

					var conn net.Conn
					Eventually(func() (err error) {
						conn, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
						return err
					}, 1*time.Second).ShouldNot(HaveOccurred())

					Eventually(func() error {
						_, err := sendData(conn, "write that should fail")
						return err
					}, proxyConfig.HealthcheckTimeout()*4, 200*time.Millisecond).Should(HaveOccurred())

				}, 20)
			})
		})
	})
})
