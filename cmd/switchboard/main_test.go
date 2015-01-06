package main_test

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
	"github.com/tedsuo/ifrit/grouper"
)

func switchboardRunner(args ...string) ifrit.Runner {
	command := exec.Command(switchboardBinPath, args...)
	runner := ginkgomon.New(ginkgomon.Config{
		Command:    command,
		Name:       fmt.Sprintf("switchboard"),
		StartCheck: "started",
	})
	return runner
}

func backendRunner(port uint) ifrit.Runner {
	command := exec.Command(dummyBackendBinPath, fmt.Sprintf("-port=%d", port))
	runner := ginkgomon.New(ginkgomon.Config{
		Command:    command,
		Name:       fmt.Sprintf("fake-backend:%d", port),
		StartCheck: "Backend listening on",
	})
	return runner
}

func healthcheckRunner(port uint) ifrit.Runner {
	command := exec.Command(dummyHealthcheckBinPath, fmt.Sprintf("-port=%d", port))
	runner := ginkgomon.New(ginkgomon.Config{
		Command:    command,
		Name:       fmt.Sprintf("fake-healthcheck:%d", port),
		StartCheck: "Healthcheck listening on",
	})
	return runner
}

func sendData(conn net.Conn, data string) (string, error) {
	conn.Write([]byte(data))
	buffer := make([]byte, 1024)
	_, err := conn.Read(buffer)
	if err != nil {
		return "", err
	} else {
		return string(buffer), nil
	}
}

var _ = Describe("Switchboard", func() {
	var process ifrit.Process

	BeforeEach(func() {
		group := grouper.NewParallel(os.Kill, grouper.Members{
			grouper.Member{"backend-1", backendRunner(backendPort)},
			grouper.Member{"backend-2", backendRunner(backendPort2)},
			grouper.Member{"healthcheck-1", healthcheckRunner(dummyHealthcheckPort)},
			grouper.Member{"healthcheck-2", healthcheckRunner(dummyHealthcheckPort2)},
			grouper.Member{"switchboard", switchboardRunner(
				fmt.Sprintf("-config=%s", proxyConfigFile),
				fmt.Sprintf("-pidFile=%s", pidFile),
			)},
		})
		process = ifrit.Invoke(group)
	})

	AfterEach(func() {
		ginkgomon.Kill(process)
	})

	Context("when there are multiple concurrent clients", func() {
		var conn1, conn2, conn3 net.Conn
		var data1, data2, data3 string

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

			Expect(data1).Should(ContainSubstring("test1"))
			Expect(data2).Should(ContainSubstring("test2"))
			Expect(data3).Should(ContainSubstring("test3"))
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
			Expect(dataBeforeDisconnect).Should(ContainSubstring("data before disconnect"))

			connToDisconnect.Close()

			dataAfterDisconnect, err := sendData(conn, "data after disconnect")
			Expect(err).ToNot(HaveOccurred())
			Expect(dataAfterDisconnect).Should(ContainSubstring("data after disconnect"))
		})
	})

	Context("when the healthcheck succeeds", func() {
		var client net.Conn

		It("checks health again after the specified interval", func() {
			Eventually(func() error {
				var err error
				client, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
				return err
			}).ShouldNot(HaveOccurred())

			buffer := make([]byte, 1024)

			client.Write([]byte("data around first healthcheck"))
			n, err := client.Read(buffer)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(buffer[:n])).Should(ContainSubstring("data around first healthcheck"))

			Consistently(func() error {
				client.Write([]byte("data around subsequent healthcheck"))
				_, err = client.Read(buffer)
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
				Expect(dataWhileHealthy).Should(ContainSubstring("data while healthy"))

				resp, httpErr := http.Get(fmt.Sprintf("http://localhost:%d/set503", dummyHealthcheckPort))
				Expect(httpErr).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				resp, httpErr = http.Get(fmt.Sprintf("http://localhost:%d/", dummyHealthcheckPort))
				Expect(resp.StatusCode).To(Equal(http.StatusServiceUnavailable))
				Expect(httpErr).NotTo(HaveOccurred())

				Eventually(func() error {
					_, err := sendData(conn, "data when unhealthy")
					return err
				}, 2*time.Second).Should(HaveOccurred())
			})
		})

		Context("when a backend goes down", func() {
			var conn net.Conn
			var data string

			BeforeEach(func() {
				Eventually(func() (err error) {
					conn, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
					return err
				}).ShouldNot(HaveOccurred())

				data, err := sendData(conn, "data before hang")
				Expect(err).ShouldNot(HaveOccurred())
				Expect(data).Should(ContainSubstring("data before hang"))

				resp, httpErr := http.Get(fmt.Sprintf("http://localhost:%d/setHang", dummyHealthcheckPort))
				Expect(httpErr).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			})

			It("disconnects existing client connections", func(done Done) {
				defer close(done)

				Eventually(func() error {
					_, err := sendData(conn, "data after hang")
					return err
				}, proxyConfig.HealthcheckTimeout()*4).Should(HaveOccurred())
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
				Expect(data).Should(ContainSubstring(fmt.Sprintf("Echo from port %d: test", backendPort2)))
			}, 5)
		})

		Context("when all backends are down", func() {
			BeforeEach(func() {
				resp, httpErr := http.Get(fmt.Sprintf("http://localhost:%d/setHang", dummyHealthcheckPort))
				Expect(httpErr).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				resp, httpErr = http.Get(fmt.Sprintf("http://localhost:%d/setHang", dummyHealthcheckPort2))
				Expect(httpErr).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
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
				}, proxyConfig.HealthcheckTimeout()*4).Should(HaveOccurred())

			}, 20)
		})
	})
})
