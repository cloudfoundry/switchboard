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

const (
	BACKEND_IP = "localhost"
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

func startHealthCheck(args ...string) *gexec.Session {
	command := exec.Command(dummyHealthCheckBinPath, args...)
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session).Should(gbytes.Say("Healthcheck listening on"))
	return session
}

var _ = Describe("Switchboard", func() {
	var (
		backendSession     *gexec.Session
		healthcheckSession *gexec.Session
		proxySession       *gexec.Session
	)

	BeforeEach(func() {
		backendSession = startBackend(
			fmt.Sprintf("-port=%d", backendPort),
		)

		healthcheckSession = startHealthCheck(
			fmt.Sprintf("-port=%d", dummyHealthCheckPort),
		)

		proxySession = startSwitchboard(
			fmt.Sprintf("-port=%d", switchboardPort),
			fmt.Sprintf("-backendIp=%s", BACKEND_IP),
			fmt.Sprintf("-backendPort=%d", backendPort),
			fmt.Sprintf("-healthcheckPort=%d", dummyHealthCheckPort),
			fmt.Sprintf("-healthcheckTimeout=%s", "500ms"),
			fmt.Sprintf("-pidfile=%s", pidfile),
		)
	})

	AfterEach(func() {
		proxySession.Terminate()
		healthcheckSession.Terminate()
		backendSession.Terminate()
	})

	Context("when there are multiple concurrent clients", func() {
		It("proxies all the connections to the backend", func() {
			count := 10

			for i := 0; i < count; i++ {
				go func(i int) {
					defer GinkgoRecover()
					var conn net.Conn
					Eventually(func() error {
						var err error
						conn, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
						return err
					}, 1*time.Second, 10*time.Millisecond).ShouldNot(HaveOccurred())

					data := make([]byte, 1024)

					conn.Write([]byte(fmt.Sprintf("test%d", i)))
					n, err := conn.Read(data)

					Expect(err).ToNot(HaveOccurred())
					Expect(string(data[:n])).Should(ContainSubstring(fmt.Sprintf("Echo: test%d", i)))
				}(i)
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
			}, 1*time.Second, 10*time.Millisecond).ShouldNot(HaveOccurred())

			Eventually(func() error {
				var err error
				connToDisconnect, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
				return err
			}, 1*time.Second, 10*time.Millisecond).ShouldNot(HaveOccurred())

			buffer := make([]byte, 1024)

			conn.Write([]byte("data before disconnect"))
			n, err := conn.Read(buffer)

			Expect(err).ToNot(HaveOccurred())
			Expect(string(buffer[:n])).Should(ContainSubstring("data before disconnect"))

			connToDisconnect.Close()

			conn.Write([]byte("data after disconnect"))
			n, err = conn.Read(buffer)

			Expect(err).ToNot(HaveOccurred())
			Expect(string(buffer[:n])).Should(ContainSubstring("data after disconnect"))
		})
	})

	Context("when the healthcheck succeeds", func() {
		var client net.Conn

		It("checks health again after the specified interval", func() {
			Eventually(func() error {
				var err error
				client, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
				return err
			}, 1*time.Second, 10*time.Millisecond).ShouldNot(HaveOccurred())

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
			}, 1*time.Second, 10*time.Millisecond).ShouldNot(HaveOccurred())

			buf := make([]byte, 1024)

			conn.Write([]byte("data1"))
			n, err := conn.Read(buf)

			Expect(err).ToNot(HaveOccurred())
			Expect(string(buf[:n])).Should(ContainSubstring("data1"))

			resp, httpErr := http.Get(fmt.Sprintf("http://localhost:%d/set503", dummyHealthCheckPort))
			Expect(httpErr).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			resp, httpErr = http.Get(fmt.Sprintf("http://localhost:%d/", dummyHealthCheckPort))
			Expect(resp.StatusCode).To(Equal(http.StatusServiceUnavailable))
			Expect(httpErr).NotTo(HaveOccurred())

			Eventually(func() error {
				conn.Write([]byte("data2"))
				_, err = conn.Read(buf)
				return err
			}, 5*time.Second).Should(HaveOccurred())
		})
	})

	Context("when the healthcheck hangs", func() {
		It("disconnects client connections", func(done Done) {
			defer close(done)
			defer GinkgoRecover()

			var conn net.Conn
			Eventually(func() error {
				var err error
				conn, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
				return err
			}, 1*time.Second, 10*time.Millisecond).ShouldNot(HaveOccurred())

			buf := make([]byte, 1024)

			conn.Write([]byte("data1"))
			n, err := conn.Read(buf)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(string(buf[:n])).Should(ContainSubstring("data1"))

			resp, httpErr := http.Get(fmt.Sprintf("http://localhost:%d/setHang", dummyHealthCheckPort))
			Expect(httpErr).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			Eventually(func() error {
				conn.Write([]byte("data2"))
				_, err = conn.Read(buf)
				return err
			}, 2*time.Second).Should(HaveOccurred())
		}, 5)
	})
})
