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

func startMainWithArgs(args ...string) *gexec.Session {
	command := exec.Command(switchboardBinPath, args...)
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session).Should(gbytes.Say("started on port"))
	return session
}

func startBackendWithArgs(args ...string) *gexec.Session {
	command := exec.Command(dummyListenerBinPath, args...)
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session).Should(gbytes.Say("Backend listening on"))
	return session
}

func startHealthCheckWithArgs(args ...string) *gexec.Session {
	command := exec.Command(dummyHealthCheckBinPath, args...)
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session).Should(gbytes.Say("Healthcheck listening on"))
	return session
}

var _ = Describe("Switchboard", func() {
	Context("with a single backend node", func() {
		It("forwards multiple client connections to the backend", func() {

			backendSession := startBackendWithArgs([]string{
				fmt.Sprintf("-port=%d", backendPort),
			}...)
			defer backendSession.Terminate()

			session := startMainWithArgs([]string{
				fmt.Sprintf("-port=%d", switchboardPort),
				fmt.Sprintf("-backendIp=%s", BACKEND_IP),
				fmt.Sprintf("-backendPort=%d", backendPort),
				fmt.Sprintf("-healthcheckPort=%d", dummyHealthCheckPort),
			}...)
			defer session.Terminate()

			healthcheckSession := startHealthCheckWithArgs(
				fmt.Sprintf("-port=%d", dummyHealthCheckPort),
			)
			defer healthcheckSession.Terminate()

			count := 10

			for i := 0; i < count; i++ {
				// Run the clients in parallel via goroutines
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

					Ω(err).ToNot(HaveOccurred())
					Ω(string(data[:n])).Should(ContainSubstring(fmt.Sprintf("Echo: test%d", i)))
				}(i)
			}
		})

		It("can maintain a long-lived connection when other clients disconnect", func() {

			backendSession := startBackendWithArgs([]string{
				fmt.Sprintf("-port=%d", backendPort),
			}...)
			defer backendSession.Terminate()

			session := startMainWithArgs([]string{
				fmt.Sprintf("-port=%d", switchboardPort),
				fmt.Sprintf("-backendIp=%s", BACKEND_IP),
				fmt.Sprintf("-backendPort=%d", backendPort),
				fmt.Sprintf("-healthcheckPort=%d", dummyHealthCheckPort),
			}...)
			defer session.Terminate()

			healthcheckSession := startHealthCheckWithArgs(
				fmt.Sprintf("-port=%d", dummyHealthCheckPort),
			)
			defer healthcheckSession.Terminate()

			var longConnection net.Conn
			var shortConnection net.Conn

			Eventually(func() error {
				var err error
				longConnection, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
				return err
			}, 1*time.Second, 10*time.Millisecond).ShouldNot(HaveOccurred())

			Eventually(func() error {
				var err error
				shortConnection, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
				return err
			}, 1*time.Second, 10*time.Millisecond).ShouldNot(HaveOccurred())

			longBuffer := make([]byte, 1024)
			shortBuffer := make([]byte, 1024)

			longConnection.Write([]byte("longdata"))
			n, err := longConnection.Read(longBuffer)

			Ω(err).ToNot(HaveOccurred())
			Ω(string(longBuffer[:n])).Should(ContainSubstring("longdata"))

			shortConnection.Write([]byte("shortdata"))
			n, err = shortConnection.Read(shortBuffer)

			Ω(err).ToNot(HaveOccurred())
			Ω(string(shortBuffer[:n])).Should(ContainSubstring("shortdata"))

			shortConnection.Close()

			longConnection.Write([]byte("longdata1"))
			n, err = longConnection.Read(longBuffer)

			Ω(err).ToNot(HaveOccurred())
			Ω(string(longBuffer[:n])).Should(ContainSubstring("longdata1"))
		})

		It("severs client connections when healthcheck reports 503", func() {
			backendSession := startBackendWithArgs(
				fmt.Sprintf("-port=%d", backendPort),
			)
			defer backendSession.Terminate()

			healthcheckSession := startHealthCheckWithArgs(
				fmt.Sprintf("-port=%d", dummyHealthCheckPort),
			)
			defer healthcheckSession.Terminate()

			proxySession := startMainWithArgs(
				fmt.Sprintf("-port=%d", switchboardPort),
				fmt.Sprintf("-backendIp=%s", BACKEND_IP),
				fmt.Sprintf("-backendPort=%d", backendPort),
				fmt.Sprintf("-healthcheckPort=%d", dummyHealthCheckPort),
			)
			defer proxySession.Terminate()

			var conn net.Conn
			Eventually(func() error {
				var err error
				conn, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
				return err
			}, 1*time.Second, 10*time.Millisecond).ShouldNot(HaveOccurred())

			buf := make([]byte, 1024)

			conn.Write([]byte("data1"))
			n, err := conn.Read(buf)

			Ω(err).ToNot(HaveOccurred())
			Ω(string(buf[:n])).Should(ContainSubstring("data1"))

			resp, httpErr := http.Get(fmt.Sprintf("http://localhost:%d/set503", dummyHealthCheckPort))
			Expect(httpErr).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			resp, httpErr = http.Get(fmt.Sprintf("http://localhost:%d/", dummyHealthCheckPort))
			Expect(resp.StatusCode).To(Equal(http.StatusServiceUnavailable))
			Expect(httpErr).NotTo(HaveOccurred())

			time.Sleep(1 * time.Second)

			conn.Write([]byte("data2"))
			n, err = conn.Read(buf)

			Ω(err).To(HaveOccurred())

		})
	})
})
