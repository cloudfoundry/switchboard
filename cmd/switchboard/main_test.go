package main_test

import (
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)


func startSwitchboard(args ...string) *gexec.Session {
	command := exec.Command(switchboardBinPath, args...)
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session).Should(gbytes.Say("started on port"))
	return session
}

func startBackend(args ...string) *gexec.Session {
	command := exec.Command(dummyListenerBinPath, args...)
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session).Should(gbytes.Say("Backend listening on"))
	return session
}

func startHealthcheck(args ...string) *gexec.Session {
	command := exec.Command(dummyHealthcheckBinPath, args...)
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session).Should(gbytes.Say("Healthcheck listening on"))
	return session
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
	var (
		backendSession     *gexec.Session
		backendSession2     *gexec.Session
		healthcheckSession *gexec.Session
		healthcheckSession2 *gexec.Session
		proxySession       *gexec.Session
		healthcheckTimeout time.Duration
	)

	BeforeEach(func() {
		healthcheckTimeout = 500 * time.Millisecond

		backendSession = startBackend(
			fmt.Sprintf("-port=%d", backendPort),
		)

		backendSession2 = startBackend(
			fmt.Sprintf("-port=%d", backendPort2),
		)

		healthcheckSession = startHealthcheck(
			fmt.Sprintf("-port=%d", dummyHealthcheckPort),
		)

		healthcheckSession2 = startHealthcheck(
			fmt.Sprintf("-port=%d", dummyHealthcheckPort2),
		)

		proxySession = startSwitchboard(
			"-backendIPs=localhost, localhost",
			fmt.Sprintf("-port=%d", switchboardPort),
			fmt.Sprintf("-backendPorts=%d,%d", backendPort, backendPort2),
			fmt.Sprintf("-healthcheckPorts=%d,%d", dummyHealthcheckPort, dummyHealthcheckPort2),
			fmt.Sprintf("-healthcheckTimeout=%s", healthcheckTimeout),
			fmt.Sprintf("-pidfile=%s", pidfile),
		)
	})

	AfterEach(func() {
		proxySession.Terminate()
		healthcheckSession2.Terminate()
		healthcheckSession.Terminate()
		backendSession2.Terminate()
		backendSession.Terminate()
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

			Expect(data1).Should(ContainSubstring(fmt.Sprintf("Echo from port %d: test1", backendPort)))
			Expect(data2).Should(ContainSubstring(fmt.Sprintf("Echo from port %d: test2", backendPort)))
			Expect(data3).Should(ContainSubstring(fmt.Sprintf("Echo from port %d: test3", backendPort)))
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

	Context("when the healthcheck hangs", func() {
		It("disconnects existing client connections", func(done Done) {
			defer close(done)
			defer GinkgoRecover()

			var conn net.Conn
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

			Eventually(func() error {
				_, err := sendData(conn, "data after hang")
				return err
			}, healthcheckTimeout*4).Should(HaveOccurred())
		}, 5)

		It("disconnects any new connections that are made", func(done Done) {
			defer close(done)
			defer GinkgoRecover()

			resp, httpErr := http.Get(fmt.Sprintf("http://localhost:%d/setHang", dummyHealthcheckPort))
			Expect(httpErr).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			time.Sleep(healthcheckTimeout)

			var conn net.Conn
			Eventually(func() (err error) {
				conn, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
				return err
			}, 1*time.Second).ShouldNot(HaveOccurred())

			Eventually(func() error {
				_, err := sendData(conn, "data after hang")
				return err
			}, healthcheckTimeout*4, healthcheckTimeout/2).Should(HaveOccurred())
		}, 5)
	})
})
