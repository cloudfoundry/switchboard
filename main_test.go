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

	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/cloudfoundry-incubator/consuladapter/consulrunner"
	"github.com/cloudfoundry-incubator/locket"
	"github.com/cloudfoundry-incubator/switchboard/config"
	"github.com/cloudfoundry-incubator/switchboard/dummies"
	"github.com/hashicorp/consul/api"
	. "github.com/onsi/ginkgo"
	ginkgoconf "github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
	"github.com/tedsuo/ifrit/grouper"
)

type Response struct {
	BackendPort  uint
	BackendIndex uint
	Message      string
}

func allowTraffic(allow bool) {
	var url string
	if allow {
		url = fmt.Sprintf(
			"http://localhost:%d/v0/cluster?trafficEnabled=%t",
			switchboardAPIPort,
			allow,
		)
	} else {
		url = fmt.Sprintf(
			"http://localhost:%d/v0/cluster?trafficEnabled=%t&message=%s",
			switchboardAPIPort,
			allow,
			"main%20test%20is%20disabling%20traffic",
		)
	}

	req, err := http.NewRequest("PATCH", url, nil)
	Expect(err).NotTo(HaveOccurred())
	req.SetBasicAuth("username", "password")

	client := &http.Client{}
	resp, err := client.Do(req)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
}

func getClusterFromAPI(req *http.Request) map[string]interface{} {
	client := &http.Client{}
	resp, err := client.Do(req)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	returnedCluster := map[string]interface{}{}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&returnedCluster)
	Expect(err).NotTo(HaveOccurred())
	return returnedCluster
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
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

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

		logLevel := "debug"
		switchboardRunner := ginkgomon.New(ginkgomon.Config{
			Command: exec.Command(
				switchboardBinPath,
				fmt.Sprintf("-configPath=%s", configPath),
				fmt.Sprintf("-logLevel=%s", logLevel),
			),
			Name:              fmt.Sprintf("switchboard"),
			StartCheck:        "started",
			StartCheckTimeout: startupTimeout,
		})

		group := grouper.NewParallel(os.Interrupt, grouper.Members{
			{Name: "backend-0", Runner: dummies.NewBackendRunner(0, backends[0])},
			{Name: "backend-1", Runner: dummies.NewBackendRunner(1, backends[1])},
			{Name: "healthcheck-0", Runner: healthcheckRunners[0]},
			{Name: "healthcheck-1", Runner: healthcheckRunners[1]},
			{Name: "switchboard", Runner: switchboardRunner},
		})
		process = ifrit.Invoke(group)
	})

	AfterEach(func() {
		ginkgomon.Kill(process)
	})

	Context("When consul is not configured", func() {
		Context("and switchboard starts successfully", func() {
			JustBeforeEach(func() {
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

			It("writes its PidFile", func() {
				finfo, err := os.Stat(pidFile)
				Expect(err).NotTo(HaveOccurred())
				Expect(finfo.Mode().Perm()).To(Equal(os.FileMode(0644)))
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
					var err error
					var conn net.Conn
					Eventually(func() error {
						conn, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", rootConfig.HealthPort))
						if err != nil {
							return err
						}
						return nil

					}, startupTimeout).Should(Succeed())
					defer conn.Close()

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

							Expect(returnedBackends[0]["trafficEnabled"]).To(BeTrue())
							Expect(returnedBackends[1]["trafficEnabled"]).To(BeTrue())

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
							var err error
							var conn net.Conn
							Eventually(func() error {
								conn, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
								if err != nil {
									return err
								}
								return nil

							}, startupTimeout).Should(Succeed())
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

			Describe("/v0/cluster", func() {
				Describe("GET", func() {
					It("returns valid JSON in body", func() {
						url := fmt.Sprintf("http://localhost:%d/v0/cluster", switchboardAPIPort)
						req, err := http.NewRequest("GET", url, nil)
						Expect(err).NotTo(HaveOccurred())
						req.SetBasicAuth("username", "password")

						returnedCluster := getClusterFromAPI(req)

						Expect(returnedCluster["trafficEnabled"]).To(BeTrue())
					})
				})

				Describe("PATCH", func() {
					It("returns valid JSON in body", func() {
						url := fmt.Sprintf("http://localhost:%d/v0/cluster?trafficEnabled=true", switchboardAPIPort)
						req, err := http.NewRequest("PATCH", url, nil)
						Expect(err).NotTo(HaveOccurred())
						req.SetBasicAuth("username", "password")

						returnedCluster := getClusterFromAPI(req)

						Expect(returnedCluster["trafficEnabled"]).To(BeTrue())
						Expect(returnedCluster["lastUpdated"]).NotTo(BeEmpty())
					})

					It("persists the provided value of enableTraffic", func() {
						url := fmt.Sprintf("http://localhost:%d/v0/cluster?trafficEnabled=false&message=some-reason", switchboardAPIPort)
						req, err := http.NewRequest("PATCH", url, nil)
						Expect(err).NotTo(HaveOccurred())
						req.SetBasicAuth("username", "password")

						returnedCluster := getClusterFromAPI(req)

						Expect(returnedCluster["trafficEnabled"]).To(BeFalse())

						url = fmt.Sprintf("http://localhost:%d/v0/cluster?trafficEnabled=true", switchboardAPIPort)
						req, err = http.NewRequest("PATCH", url, nil)
						Expect(err).NotTo(HaveOccurred())
						req.SetBasicAuth("username", "password")

						returnedCluster = getClusterFromAPI(req)

						Expect(returnedCluster["trafficEnabled"]).To(BeTrue())
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
								}, startupTimeout).ShouldNot(HaveOccurred())

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
						}, startupTimeout).Should(Succeed())

						Eventually(func() error {
							var err error
							connToDisconnect, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
							return err
						}, "5s").Should(Succeed())

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
						}, startupTimeout).Should(Succeed())

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
							}, startupTimeout).Should(Succeed())

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
							}, startupTimeout).Should(Succeed())

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

				Context("when traffic is disabled", func() {
					It("disconnects client connections", func() {
						var conn net.Conn
						Eventually(func() error {
							var err error
							conn, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
							return err
						}, startupTimeout).Should(Succeed())

						dataWhileHealthy, err := sendData(conn, "data while healthy")
						Expect(err).ToNot(HaveOccurred())
						Expect(dataWhileHealthy.Message).To(Equal("data while healthy"))

						allowTraffic(false)

						Eventually(func() error {
							_, err := sendData(conn, "data when unhealthy")
							return err
						}, healthcheckWaitDuration).Should(matchConnectionDisconnect())
					})

					It("rejects new connections", func() {
						Eventually(func() error {
							allowTraffic(false)

							conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
							if err != nil {
								return err
							}
							_, err = sendData(conn, "write that should fail")
							return err
						}, healthcheckWaitDuration, 200*time.Millisecond).Should(matchConnectionDisconnect())
					})

					It("permits new connections again after re-enabling traffic", func() {
						allowTraffic(false)
						allowTraffic(true)

						Eventually(func() error {
							var err error
							_, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
							return err
						}, "5s").Should(Succeed())
					})
				})
			})
		})

		Context("and switchboard is failing", func() {
			BeforeEach(func() {
				rootConfig.StaticDir = "this is totallly invalid so switchboard won't start"
			})

			It("does not write the PidFile", func() {
				Consistently(func() error {
					_, err := os.Stat(pidFile)
					return err
				}).Should(HaveOccurred())
			})
		})
	})

	Describe("consul", func() {
		var (
			consulRunner *consulrunner.ClusterRunner
			consulClient consuladapter.Client
		)

		BeforeEach(func() {
			consulRunner = consulrunner.NewClusterRunner(
				9001+ginkgoconf.GinkgoConfig.ParallelNode*consulrunner.PortOffsetLength,
				1,
				"http",
			)

			consulRunner.Start()
			consulRunner.WaitUntilReady()
			rootConfig.ConsulCluster = consulRunner.ConsulCluster()
			rootConfig.ConsulServiceName = "test_mysql"
			consulClient = consulRunner.NewClient()
		})

		AfterEach(func() {
			consulRunner.Reset()
			consulRunner.Stop()
			rootConfig.ConsulCluster = ""
		})

		It("immediately writes its PidFile", func() {
			Eventually(func() os.FileMode {
				finfo, err := os.Stat(pidFile)
				if err != nil {
					return 0
				}
				return finfo.Mode().Perm()
			}, startupTimeout).Should(Equal(os.FileMode(0644)))
		})

		Context("the switchboard acquires the lock", func() {
			It("starts up", func() {
				Eventually(func() error {
					_, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
					return err
				}, startupTimeout).Should(Succeed())
			})

			It("registers itself with consul", func() {
				Eventually(func() map[string]*api.AgentService {
					services, _ := consulClient.Agent().Services()
					return services
				}, startupTimeout).Should(HaveKeyWithValue("test_mysql",
					&api.AgentService{
						Service: "test_mysql",
						ID:      "test_mysql",
						Port:    int(switchboardPort),
					}))
			})

			Context("but then loses the lock", func() {
				It("exits with an error", func() {
					processErr := make(chan error)
					go func() { err := <-process.Wait(); processErr <- err }()

					consulRunner.Reset()
					err := <-processErr
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("and the lock is not available", func() {
			var competingSwitchboardLockProcess ifrit.Process
			BeforeEach(func() {
				logger := lagertest.NewTestLogger("test")

				competingSwitchboardLock := locket.NewLock(logger, consulClient, locket.LockSchemaPath("test_mysql_lock"), []byte{}, clock.NewClock(), time.Millisecond*2500, time.Second*5)
				competingSwitchboardLockProcess = ifrit.Invoke(competingSwitchboardLock)
			})

			AfterEach(func() {
				ginkgomon.Kill(competingSwitchboardLockProcess)
			})

			It("waits for the lock to become available", func() {
				Consistently(func() error {
					_, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
					return err
				}).Should(HaveOccurred())
			})
		})
	})
})
