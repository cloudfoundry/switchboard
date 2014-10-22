package main_test

import (
	"fmt"
	"net"
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
	fmt.Printf("Switchboard started with args:%v\n", args)
	return session
}

func startBackendWithArgs(args ...string) *gexec.Session {
	command := exec.Command(dummyListenerBinPath, args...)
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session).Should(gbytes.Say("Backend listening on"))
	fmt.Printf("Backend started with args:%v\n", args)
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
			}...)
			defer session.Terminate()

			count := 10
			buffers := make([][]byte, count)
			conns := make([]net.Conn, count)

			for i := 0; i < count; i++ {
				var conn net.Conn
				Eventually(func() error {
					var err error
					conn, err = net.Dial("tcp", fmt.Sprintf("localhost:%d", switchboardPort))
					return err
				}, 1*time.Second, 10*time.Millisecond).ShouldNot(HaveOccurred())

				buffers[i] = make([]byte, 1024)
				conns[i] = conn
			}

			for i, conn := range conns {
				// Run the clients in parallel via goroutines
				go func(i int, conn net.Conn) {
					data := buffers[i]
					var n int
					var err error

					conn.Write([]byte(fmt.Sprintf("test%d", i)))
					// Read is a blocking method so we background it in a goroutine.
					go func() {
						n, err = conn.Read(data)
					}()

					// Read is asynchronous so we need to use Eventually
					Eventually(func() string {
						return string(data[:n])
					}).Should(ContainSubstring(fmt.Sprintf("Echo: test%d", i)))
				}(i, conn)
			}
		})
	})
})
