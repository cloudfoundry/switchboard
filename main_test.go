package main_test

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/cloudfoundry-incubator/switchboard/config"
	"github.com/cloudfoundry-incubator/switchboard/dummies"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
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
		return Response{}, err.(error)
	} else {
		response := Response{}
		err := json.Unmarshal(buffer[:n], &response)
		if err != nil {
			return Response{}, err.(error)
		}
		return response, nil
	}
}

func verifyHeaderContains(header http.Header, key, valueSubstring string) {
	found := false
	for k, v := range header {
		if k == key {
			for _, value := range v {
				if strings.Contains(value, valueSubstring) {
					found = true
				}
			}
		}
	}
	Expect(found).To(BeTrue(), fmt.Sprintf("%s: %s not found in header", key, valueSubstring))
}

func getBackendsFromApi(req *http.Request) []map[string]interface{} {
	client := &http.Client{}
	resp, err := client.Do(req)
	Expect(err).NotTo(HaveOccurred())

	returnedBackends := []map[string]interface{}{}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&returnedBackends)
	Expect(err).NotTo(HaveOccurred())
	return returnedBackends
}

func matchConnectionDisconnect() types.GomegaMatcher {
	//exact error depends on environment
	return MatchError(
		MatchRegexp(
			"%s|%s",
			io.EOF.Error(),
			syscall.ECONNRESET.Error(),
		),
	)
}

var _ = Describe("Switchboard", func() {
	var process ifrit.Process
	var initialActiveBackend, initialInactiveBackend config.Backend
	var healthcheckRunners []*dummies.HealthcheckRunner
	var healthcheckWaitDuration time.Duration
	const startupTimeout = 10 * time.Second

	BeforeEach(func() {
		initConfig()
		healthcheckWaitDuration = 3 * proxyConfig.HealthcheckTimeout()
	})

	JustBeforeEach(func() {
		writeConfig()

		healthcheckRunners = []*dummies.HealthcheckRunner{
			dummies.NewHealthcheckRunner(backends[0]),
			dummies.NewHealthcheckRunner(backends[1]),
		}

		switchboardRunner := ginkgomon.New(ginkgomon.Config{
			Command: exec.Command(
				switchboardBinPath,
				fmt.Sprintf("-configPath=%s", configPath),
				fmt.Sprintf("-pidFile=%s", pidFile),
				fmt.Sprintf("-staticDir=%s", staticDir),
			),
			Name:              fmt.Sprintf("switchboard"),
			StartCheck:        "started",
			StartCheckTimeout: startupTimeout,
		})

		group := grouper.NewParallel(os.Kill, grouper.Members{
			{Name: "backend-0", Runner: dummies.NewBackendRunner(0, backends[0])},
			{Name: "backend-1", Runner: dummies.NewBackendRunner(1, backends[1])},
			{Name: "healthcheck-0", Runner: healthcheckRunners[0]},
			{Name: "healthcheck-1", Runner: healthcheckRunners[1]},
			{Name: "switchboard", Runner: switchboardRunner},
		})
		process = ifrit.Invoke(group)

		var err error
		var conn net.Conn
		Eventually(func() error {
			conn, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
			return err
		}, startupTimeout).Should(Succeed())
		defer conn.Close()

		response, err := sendData(conn, "detect active")
		Expect(err).NotTo(HaveOccurred())

		initialActiveBackend = backends[response.BackendIndex]
		initialInactiveBackend = backends[(response.BackendIndex+1)%2]
	})

	AfterEach(func() {
		ginkgomon.Kill(process)
	})

	Describe("Profiler", func() {
		It("responds with 200 at /debug/pprof", func() {
			url := fmt.Sprintf("http://localhost:%d/debug/pprof/", switchboardProfilerPort)
			resp, err := http.Get(url)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	})

	Describe("Health", func() {
		var acceptsAndClosesTCPConnections = func() {
			conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", rootConfig.HealthPort))
			Expect(err).NotTo(HaveOccurred())

			err = conn.Close()
			Expect(err).NotTo(HaveOccurred())
		}

		It("accepts and immediately closes TCP connections on HealthPort", func() {
			acceptsAndClosesTCPConnections()
		})

		Context("when HealthPort == API.Port", func() {
			BeforeEach(func() {
				rootConfig.HealthPort = rootConfig.API.Port
			})

			It("operates normally", func() {
				acceptsAndClosesTCPConnections()
			})
		})
	})

	Describe("UI", func() {
		Describe("/", func() {
			var url string

			BeforeEach(func() {
				url = fmt.Sprintf("http://localhost:%d/", switchboardAPIPort)
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

			It("responds with 200 and contains non-zero body when authorized", func() {
				req, err := http.NewRequest("GET", url, nil)
				req.SetBasicAuth("username", "password")
				client := &http.Client{}
				resp, err := client.Do(req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				Expect(resp.Body).ToNot(BeNil())
				defer resp.Body.Close()
				body, err := ioutil.ReadAll(resp.Body)
				Expect(len(body)).To(BeNumerically(">", 0), "Expected body to not be empty")
			})
		})
	})

	Describe("api", func() {
		Describe("/v0/backends/", func() {
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

			Context("When authorized", func() {
				var req *http.Request

				BeforeEach(func() {
					var err error
					req, err = http.NewRequest("GET", url, nil)
					Expect(err).NotTo(HaveOccurred())
					req.SetBasicAuth("username", "password")
				})

				It("returns correct headers", func() {
					client := &http.Client{}
					resp, err := client.Do(req)
					Expect(err).NotTo(HaveOccurred())
					Expect(resp.StatusCode).To(Equal(http.StatusOK))
					verifyHeaderContains(resp.Header, "Content-Type", "application/json")
				})

				It("returns valid JSON in body", func() {

					returnedBackends := getBackendsFromApi(req)

					Expect(len(returnedBackends)).To(Equal(2))

					Expect(returnedBackends[0]["host"]).To(Equal("localhost"))
					Expect(returnedBackends[0]["healthy"]).To(BeTrue(), "Expected backends[0] to be healthy")

					Expect(returnedBackends[1]["host"]).To(Equal("localhost"))
					Expect(returnedBackends[1]["healthy"]).To(BeTrue(), "Expected backends[1] to be healthy")

					switch returnedBackends[0]["name"] {

					case backends[0].Name:
						Expect(returnedBackends[0]["port"]).To(BeNumerically("==", backends[0].Port))
						Expect(returnedBackends[1]["port"]).To(BeNumerically("==", backends[1].Port))
						Expect(returnedBackends[1]["name"]).To(Equal(backends[1].Name))

					case backends[1].Name: // order reversed in response
						Expect(returnedBackends[1]["port"]).To(BeNumerically("==", backends[0].Port))
						Expect(returnedBackends[0]["port"]).To(BeNumerically("==", backends[1].Port))
						Expect(returnedBackends[0]["name"]).To(Equal(backends[1].Name))
					default:
						Fail(fmt.Sprintf("Invalid backend name: %s", returnedBackends[0]["name"]))
					}
				})

				It("returns session count for active and inactive backends", func() {

					conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
					Expect(err).ToNot(HaveOccurred())
					defer conn.Close()

					connData, err := sendData(conn, "success")
					Expect(err).ToNot(HaveOccurred())
					Expect(connData.Message).To(Equal("success"))

					returnedBackends := getBackendsFromApi(req)

					var activeBackend, inactiveBackend map[string]interface{}
					if returnedBackends[0]["active"].(bool) {
						activeBackend = returnedBackends[0]
						inactiveBackend = returnedBackends[1]
					} else {
						activeBackend = returnedBackends[1]
						inactiveBackend = returnedBackends[0]
					}

					Expect(activeBackend["currentSessionCount"]).To(BeNumerically("==", 1), "Expected active backend to have SessionCount == 1")
					Expect(inactiveBackend["currentSessionCount"]).To(BeNumerically("==", 0), "Expected inactive backend to have SessionCount == 0")
					Expect(inactiveBackend["active"]).To(BeFalse(), "Expected inactive backend to not be active")
				})
			})
		})
	})

	Describe("proxy", func() {
		Context("when there are multiple concurrent clients", func() {

			It("proxies all the connections to the backend", func() {

				var doneArray = make([]chan interface{}, 3)
				var dataMessages = make([]string, 3)

				for i := 0; i < 3; i++ {
					doneArray[i] = make(chan interface{})
					go func(index int) {
						defer GinkgoRecover()
						defer close(doneArray[index])

						var err error
						var conn net.Conn

						Eventually(func() error {
							conn, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
							return err
						}).ShouldNot(HaveOccurred())

						data, err := sendData(conn, fmt.Sprintf("test%d", index))
						Expect(err).ToNot(HaveOccurred())
						dataMessages[index] = data.Message
					}(i)
				}

				for _, done := range doneArray {
					<-done
				}

				for i, message := range dataMessages {
					Expect(message).Should(Equal(fmt.Sprintf("test%d", i)))
				}
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
				}).Should(Succeed())

				Eventually(func() error {
					var err error
					connToDisconnect, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
					return err
				}).Should(Succeed())

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
				}).Should(Succeed())

				data, err := sendData(client, "data around first healthcheck")
				Expect(err).NotTo(HaveOccurred())
				Expect(data.Message).To(Equal("data around first healthcheck"))

				Consistently(func() error {
					_, err = sendData(client, "data around subsequent healthcheck")
					return err
				}, 3*time.Second, 500*time.Millisecond).Should(Succeed())
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
					}).Should(Succeed())

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
					}, healthcheckWaitDuration).Should(matchConnectionDisconnect())
				})
			})

			Context("when a backend goes down", func() {
				var conn net.Conn
				var data Response

				JustBeforeEach(func() {
					Eventually(func() (err error) {
						conn, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
						return err
					}).Should(Succeed())

					data, err := sendData(conn, "data before hang")
					Expect(err).ToNot(HaveOccurred())
					Expect(data.Message).To(Equal("data before hang"))

					if initialActiveBackend == backends[0] {
						healthcheckRunners[0].SetHang(true)
					} else {
						healthcheckRunners[1].SetHang(true)
					}
				})

				It("disconnects existing client connections", func() {
					Eventually(func() error {
						_, err := sendData(conn, "data after hang")
						return err
					}, healthcheckWaitDuration).Should(matchConnectionDisconnect())
				})

				It("proxies new connections to another backend", func() {
					var err error
					Eventually(func() (uint, error) {
						conn, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
						if err != nil {
							return 0, err
						}

						data, err = sendData(conn, "test")
						return data.BackendPort, err
					}, healthcheckWaitDuration).Should(Equal(initialInactiveBackend.Port))

					Expect(data.Message).To(Equal("test"))
				})
			})

			Context("when all backends are down", func() {
				JustBeforeEach(func() {
					for _, hr := range healthcheckRunners {
						hr.SetHang(true)
					}
				})

				It("rejects any new connections that are attempted", func() {

					Eventually(func() error {
						conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
						if err != nil {
							return err
						}
						_, err = sendData(conn, "write that should fail")
						return err
					}, healthcheckWaitDuration, 200*time.Millisecond).Should(matchConnectionDisconnect())
				})
			})
		})
	})
})
