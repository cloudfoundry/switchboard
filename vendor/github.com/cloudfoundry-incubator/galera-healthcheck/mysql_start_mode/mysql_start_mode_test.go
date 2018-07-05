package mysql_start_mode_test

import (
	"io/ioutil"
	"os"

	"github.com/cloudfoundry-incubator/galera-healthcheck/mysql_start_mode"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("GaleraStartMySQL", func() {
	var stateFile *os.File

	Context("accepts a parameter for the type of startup it will do", func() {
		BeforeEach(func() {
			stateFile, _ = ioutil.TempFile(os.TempDir(), "stateFile")
			stateFile.Chmod(0777)
		})

		AfterEach(func() {
			os.Remove(stateFile.Name())
		})

		Context("bootstrap mode", func() {
			It("is passed a 'bootstrap' parameter", func() {
				mysqlStartMode := mysql_start_mode.NewMysqlStartMode(stateFile.Name(), "bootstrap")
				err := mysqlStartMode.Start()
				Expect(err).ToNot(HaveOccurred())
			})

			It("writes 'NEEDS_CLUSTER' to its state file", func() {
				mysqlStartMode := mysql_start_mode.NewMysqlStartMode(stateFile.Name(), "bootstrap")
				err := mysqlStartMode.Start()
				Expect(err).ToNot(HaveOccurred())
				stateFileOutput, _ := ioutil.ReadFile(stateFile.Name())
				Expect(string(stateFileOutput)).To(Equal("NEEDS_BOOTSTRAP"))
			})
		})

		Context("join mode", func() {
			It("is passed a 'join' parameter", func() {
				mysqlStartMode := mysql_start_mode.NewMysqlStartMode(stateFile.Name(), "join")
				err := mysqlStartMode.Start()
				Expect(err).ToNot(HaveOccurred())
			})

			It("writes 'NEEDS_BOOTSTRAP' to its state file", func() {
				mysqlStartMode := mysql_start_mode.NewMysqlStartMode(stateFile.Name(), "join")
				err := mysqlStartMode.Start()
				Expect(err).ToNot(HaveOccurred())
				stateFileOutput, _ := ioutil.ReadFile(stateFile.Name())
				Expect(string(stateFileOutput)).To(Equal("CLUSTERED"))
			})
		})

		Context("singleNode mode", func() {
			It("writes 'SINGLE_NODE' to its state file", func() {
				mysqlStartMode := mysql_start_mode.NewMysqlStartMode(stateFile.Name(), "singleNode")
				err := mysqlStartMode.Start()
				Expect(err).ToNot(HaveOccurred())
				stateFileOutput, _ := ioutil.ReadFile(stateFile.Name())
				Expect(string(stateFileOutput)).To(Equal("SINGLE_NODE"))
			})
		})

		It("is passed an unrecognized parameter", func() {
			mysqlStartMode := mysql_start_mode.NewMysqlStartMode("stateFileExample.txt", "not_legit_parameter")
			err := mysqlStartMode.Start()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unrecognized value for start mode"))
		})
	})
})
