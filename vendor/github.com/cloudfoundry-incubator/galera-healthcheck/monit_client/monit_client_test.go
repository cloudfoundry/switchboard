package monit_client_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	"github.com/cloudfoundry-incubator/galera-healthcheck/config"
	"github.com/cloudfoundry-incubator/galera-healthcheck/monit_client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	stateFile            *os.File
	fakeBootstrapLogFile *os.File
)

var _ = Describe("monitClient", func() {
	var (
		monitClient             *monit_client.MonitClient
		ts                      *httptest.Server
		logger                  lager.Logger
		fakeHandler             http.HandlerFunc
		processName             string
		blankBootstrapFile      bool
		enableSstMarkerFilePath string
	)

	BeforeEach(func() {
		blankBootstrapFile = false
	})

	JustBeforeEach(func() {
		ts = httptest.NewServer(fakeHandler)
		testHost, testPort := splitHostandPort(ts.URL)
		fakeBootstrapFileName := ""
		if !blankBootstrapFile {
			fakeBootstrapFileName = "fixtures/fake_prestart_script"
		}

		monitConfig := config.MonitConfig{
			User:                    "fake-user",
			Password:                "fake-password",
			Host:                    testHost,
			Port:                    testPort,
			MysqlStateFilePath:      stateFile.Name(),
			ServiceName:             processName,
			BootstrapFilePath:       fakeBootstrapFileName,
			BootstrapLogFilePath:    fakeBootstrapLogFile.Name(),
			EnableSstMarkerFilePath: enableSstMarkerFilePath,
			SstInterruptNotifyCmd:   "fake-notify-cmd",
		}

		logger = lagertest.NewTestLogger("monit_client")

		monitClient = monit_client.New(monitConfig, logger)
	})

	AfterEach(func() {
		ts.Close()
		os.Remove(stateFile.Name())
	})

	Context("when running on a mysql node", func() {
		BeforeEach(func() {
			stateFile, _ = ioutil.TempFile(os.TempDir(), "stateFile")
			stateFile.Chmod(0777)

			fakeBootstrapLogFile, _ = ioutil.TempFile(os.TempDir(), "fakeLogFile")

			fakeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			processName = "mariadb_ctrl"
		})

		Describe("StopService", func() {
			Context("when monit returns successful stop response", func() {
				BeforeEach(func() {
					fakeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						fmt.Fprintln(w, "not monitored - stop pending")
					})
				})

				It("returns http response 200 and process has stopped", func() {
					st, err := monitClient.StopService(createReq())
					Expect(err).ToNot(HaveOccurred())
					Expect(st).To(ContainSubstring("stop"))
				})
			})

			Context("when monit returns 500 error", func() {
				BeforeEach(func() {
					fakeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusInternalServerError)
						fmt.Fprintln(w, "fake-internal-error")
					})
				})

				It("returns http response non-200 and process has not stopped", func() {
					_, err := monitClient.StopService(createReq())
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-internal-error"))
				})
			})

			Context("when monit returns 200 response, but process is still running", func() {
				BeforeEach(func() {
					fakeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						fmt.Fprintln(w, "running")
					})
				})

				It("returns http response 200 and process has not stopped", func() {
					_, err := monitClient.StopService(createReq())
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("failed to stop"))
				})
			})
		})

		Describe("StartService", func() {

			Context("when monit returns 500 error", func() {

				BeforeEach(func() {
					fakeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusInternalServerError)
						fmt.Fprintln(w, "fake-internal-error")
					})
				})

				It("returns http response non-200 and process has not started", func() {
					_, err := monitClient.StartServiceJoin(createReq())
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-internal-error"))
				})
			})

			Context("when monit returns successful starting response", func() {
				BeforeEach(func() {
					fakeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						fmt.Fprintln(w, "not monitored - start pending")
					})
				})

				It("returns http response 200 and process has started", func() {
					st, err := monitClient.StartServiceJoin(createReq())
					Expect(err).ToNot(HaveOccurred())
					Expect(st).To(ContainSubstring("join"))
				})
			})

			Context("when monit returns 200 response, but process is still unstarted", func() {
				BeforeEach(func() {
					fakeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						fmt.Fprintln(w, "not monitored")
					})
				})

				It("returns http response 200 and process has not started", func() {
					_, err := monitClient.StartServiceJoin(createReq())
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("failed to start"))
				})
			})

			Context("when starting in singleNode mode", func() {
				BeforeEach(func() {
					fakeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						fmt.Fprintln(w, "not monitored - start pending")
					})
				})

				It("returns string noting successful start in singleNode mode", func() {
					st, err := monitClient.StartServiceSingleNode(createReq())
					Expect(err).ToNot(HaveOccurred())
					Expect(st).To(ContainSubstring("singleNode"))
				})
			})

			Context("when BootstrapFilePath is blank", func() {
				BeforeEach(func() {
					enableSstFile, _ := ioutil.TempFile(os.TempDir(), "EnableSstMarkerFilePath")
					enableSstFile.Close()
					enableSstMarkerFilePath = enableSstFile.Name()
					blankBootstrapFile = true
				})

				It("does not call the bootstrap binary", func() {
					monitClient.StartServiceJoin(createReq())
					logContent, err := ioutil.ReadAll(fakeBootstrapLogFile)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(logContent)).To(BeEmpty())
				})

				Context("when requester wants to enable SSTs", func() {
					It("writes to the EnableSstMarkerFilePath", func() {
						os.Remove(enableSstMarkerFilePath)
						monitClient.StartServiceJoin(createReqSSTEnabled())
						Expect(enableSstMarkerFilePath).To(BeARegularFile())
					})
				})

				Context("when requester wants to disable SSTs", func() {
					It("removes the EnableSstMarkerFilePath", func() {
						monitClient.StartServiceJoin(createReq())
						Expect(enableSstMarkerFilePath).ToNot(BeARegularFile())
					})
				})
			})

			Context("when BootstrapFilePath is set", func() {
				It("calls the bootstrap binary with SST disabled by default", func() {
					monitClient.StartServiceJoin(createReq())
					logContent, err := ioutil.ReadAll(fakeBootstrapLogFile)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(logContent)).To(ContainSubstring("SST is disabled"))
					Expect(string(logContent)).To(ContainSubstring("INTERRUPT_NOTIFY_CMD=fake-notify-cmd"))
				})
			})

			Context("when requester wants to enable SSTs", func() {
				It("calls the bootstrap binary with SST not disabled", func() {
					monitClient.StartServiceJoin(createReqSSTEnabled())
					logContent, err := ioutil.ReadAll(fakeBootstrapLogFile)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(logContent)).To(ContainSubstring("SST is enabled"))
				})
			})

			Context("when requester wants to disable SSTs", func() {
				It("calls the bootstrap binary with SST disabled", func() {
					monitClient.StartServiceJoin(createReqSSTDisabled())
					logContent, err := ioutil.ReadAll(fakeBootstrapLogFile)
					Expect(err).ToNot(HaveOccurred())
					Expect(string(logContent)).To(ContainSubstring("SST is disabled"))
					Expect(string(logContent)).To(ContainSubstring("INTERRUPT_NOTIFY_CMD=fake-notify-cmd"))
				})
			})
		})

		Describe("Status", func() {

			Context("when monit returns a valid XML response", func() {
				BeforeEach(func() {
					fixture := getRelativeFile("fixtures/monit_status.xml")
					xmlFile, err := os.Open(fixture)
					Expect(err).ToNot(HaveOccurred())
					defer xmlFile.Close()

					xmlContents, err := ioutil.ReadAll(xmlFile)
					Expect(err).ToNot(HaveOccurred())

					fakeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						w.Write(xmlContents)
					})
				})

				Context("and process is running", func() {

					BeforeEach(func() {
						processName = "running_process"
					})

					It("returns running", func() {
						stat, err := monitClient.GetStatus(createReq())
						Expect(err).ToNot(HaveOccurred())
						Expect(stat).To(Equal("running"))
					})
				})

				Context("and process is stopped", func() {
					BeforeEach(func() {
						processName = "unmonitored_process"
					})

					It("returns stopped", func() {
						stat, err := monitClient.GetStatus(createReq())
						Expect(err).ToNot(HaveOccurred())
						Expect(stat).To(Equal("stopped"))
					})
				})

				Context("and process is failing", func() {
					BeforeEach(func() {
						processName = "failing_process"
					})
					It("returns failing", func() {
						stat, err := monitClient.GetStatus(createReq())
						Expect(err).ToNot(HaveOccurred())
						Expect(stat).To(Equal("failing"))
					})
				})

				Context("and process is pending", func() {
					BeforeEach(func() {
						processName = "pending_process"
					})
					It("returns failing", func() {
						stat, err := monitClient.GetStatus(createReq())
						Expect(err).ToNot(HaveOccurred())
						Expect(stat).To(Equal("pending"))
					})
				})

				Context("and process name is not found", func() {
					BeforeEach(func() {
						processName = "nonexistent_process"
					})
					It("returns an error", func() {
						_, err := monitClient.GetStatus(createReq())
						Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Could not find process %s", processName)))
					})
				})
			})

			Context("when monit returns invalid XML", func() {
				BeforeEach(func() {
					fakeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, "not-valid-xml")
					})
				})

				It("returns an error", func() {
					_, err := monitClient.GetStatus(createReq())
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Failed to unmarshal the xml"))
				})
			})
		})
	})

	Context("when running on a arbitrator node", func() {
		BeforeEach(func() {
			stateFile, _ = ioutil.TempFile(os.TempDir(), "stateFile")
			stateFile.Chmod(0777)
			fakeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			processName = "garbd"

			fakeBootstrapLogFile, _ = ioutil.TempFile(os.TempDir(), "fakeLogFile")

		})

		Describe("StopService", func() {

			Context("when monit returns successful stop response", func() {
				BeforeEach(func() {
					fakeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						fmt.Fprintln(w, "not monitored - stop pending")
					})
				})

				It("returns http response 200 and process has stopped", func() {
					st, err := monitClient.StopService(createReq())
					Expect(err).ToNot(HaveOccurred())
					Expect(st).To(ContainSubstring("stop"))
				})
			})

			Context("when monit returns 500 error", func() {
				BeforeEach(func() {
					fakeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusInternalServerError)
						fmt.Fprintln(w, "fake-internal-error")
					})
				})

				It("returns http response non-200 and process has not stopped", func() {
					_, err := monitClient.StopService(createReq())
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-internal-error"))
				})
			})

			Context("when monit returns 200 response, but process is still running", func() {
				BeforeEach(func() {
					fakeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						fmt.Fprintln(w, "running")
					})
				})

				It("returns http response 200 and process has not stopped", func() {
					_, err := monitClient.StopService(createReq())
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("failed to stop"))
				})
			})
		})

		Describe("StartService", func() {

			Context("when monit returns 500 error", func() {

				BeforeEach(func() {
					fakeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusInternalServerError)
						fmt.Fprintln(w, "fake-internal-error")
					})
				})

				It("returns http response non-200 and process has not started", func() {
					_, err := monitClient.StartServiceJoin(createReq())
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-internal-error"))

					f, err := os.Open(stateFile.Name())
					fstat, err := f.Stat()
					Expect(int(fstat.Size())).To(Equal(0))
				})
			})

			Context("when monit returns successful starting response", func() {
				BeforeEach(func() {
					fakeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						fmt.Fprintln(w, "not monitored - start pending")
					})
				})

				It("returns http response 200 and process has started", func() {
					st, err := monitClient.StartServiceJoin(createReq())
					Expect(err).ToNot(HaveOccurred())
					Expect(st).To(ContainSubstring("join"))

					f, err := os.Open(stateFile.Name())
					fstat, err := f.Stat()
					Expect(int(fstat.Size())).To(Equal(0))
				})
			})

			Context("when monit returns 200 response, but process is still unstarted", func() {
				BeforeEach(func() {
					fakeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						fmt.Fprintln(w, "not monitored")
					})
				})

				It("returns http response 200 and process has not started", func() {
					_, err := monitClient.StartServiceJoin(createReq())
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("failed to start"))

					f, err := os.Open(stateFile.Name())
					fstat, err := f.Stat()
					Expect(int(fstat.Size())).To(Equal(0))
				})
			})

			Context("when trying to bootstrap the arbitrator node", func() {
				It("returns a message saying not allowed", func() {
					_, err := monitClient.StartServiceBootstrap(createReq())
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("bootstrapping arbitrator not allowed"))
				})
			})
		})

		Describe("Status", func() {

			Context("when monit returns a valid XML response", func() {
				BeforeEach(func() {
					fixture := getRelativeFile("fixtures/monit_status.xml")
					xmlFile, err := os.Open(fixture)
					Expect(err).ToNot(HaveOccurred())
					defer xmlFile.Close()

					xmlContents, err := ioutil.ReadAll(xmlFile)
					Expect(err).ToNot(HaveOccurred())

					fakeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						w.Write(xmlContents)
					})
				})

				Context("and process is running", func() {

					BeforeEach(func() {
						processName = "running_process"
					})

					It("returns running", func() {
						stat, err := monitClient.GetStatus(createReq())
						Expect(err).ToNot(HaveOccurred())
						Expect(stat).To(Equal("running"))
					})
				})

				Context("and process is stopped", func() {
					BeforeEach(func() {
						processName = "unmonitored_process"
					})

					It("returns stopped", func() {
						stat, err := monitClient.GetStatus(createReq())
						Expect(err).ToNot(HaveOccurred())
						Expect(stat).To(Equal("stopped"))
					})
				})

				Context("and process is failing", func() {
					BeforeEach(func() {
						processName = "failing_process"
					})
					It("returns failing", func() {
						stat, err := monitClient.GetStatus(createReq())
						Expect(err).ToNot(HaveOccurred())
						Expect(stat).To(Equal("failing"))
					})
				})

				Context("and process is pending", func() {
					BeforeEach(func() {
						processName = "pending_process"
					})
					It("returns failing", func() {
						stat, err := monitClient.GetStatus(createReq())
						Expect(err).ToNot(HaveOccurred())
						Expect(stat).To(Equal("pending"))
					})
				})

				Context("and process name is not found", func() {
					BeforeEach(func() {
						processName = "nonexistent_process"
					})
					It("returns an error", func() {
						_, err := monitClient.GetStatus(createReq())
						Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("Could not find process %s", processName)))
					})
				})
			})

			Context("when monit returns invalid XML", func() {
				BeforeEach(func() {
					fakeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						fmt.Fprint(w, "not-valid-xml")
					})
				})

				It("returns an error", func() {
					_, err := monitClient.GetStatus(createReq())
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Failed to unmarshal the xml"))
				})
			})
		})
	})
})

func splitHostandPort(url string) (string, int) {
	urlparts := strings.Split(url, ":")
	host := strings.TrimPrefix(urlparts[1], "//")
	port, _ := strconv.Atoi(urlparts[2])
	return host, port
}

func getRelativeFile(relativeFilepath string) string {
	_, filename, _, _ := runtime.Caller(1)
	thisDir := filepath.Dir(filename)
	return filepath.Join(thisDir, relativeFilepath)
}

func createReq() *http.Request {
	req, err := http.NewRequest("", "/example.com", nil)
	Expect(err).ToNot(HaveOccurred())
	return req
}

func createReqSSTDisabled() *http.Request {
	req, err := http.NewRequest("", "/example.com?sst=false", nil)
	Expect(err).ToNot(HaveOccurred())
	return req
}

func createReqSSTEnabled() *http.Request {
	req, err := http.NewRequest("", "/example.com?sst=true", nil)
	Expect(err).ToNot(HaveOccurred())
	return req
}
