package service_config_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"
)

const configJSON = `{
    "Name": "Enterprise",
    "ID": 1701,
    "Crew": {
        "Officers": [
            {"Name": "Kirk", "Role": "Commanding Officer"},
            {"Name": "Spock", "Role": "First Officer/Science Officer"},
            {"Name": "McCoy", "Role": "Chief Medical Officer"}
        ],
        "Passengers": [
            {"Name": "Sarek", "Title": "Federation Ambassador"}
        ]
    }
}`

const configYAML = `---
Name: Enterprise
ID: 1701
Crew:
  Officers:
  - {Name: Kirk, Role: Commanding Officer}
  - {Name: Spock, Role: First Officer/Science Officer}
  - {Name: McCoy, Role: Chief Medical Officer}
  Passengers:
  - {Name: Sarek, Title: Federation Ambassador}
`

const configYAMLOneLine = `{Name: Enterprise, ID: 1701, Crew: {Officers: [{Name: Kirk, Role: Commanding Officer}, {Name: Spock, Role: First Officer/Science Officer}, {Name: McCoy, Role: Chief Medical Officer}], Passengers: [{Name: Sarek, Title: Federation Ambassador}]}}`

const configJSONAlt = `{
    "Name": "Defiant",
    "ID": 74205,
    "Crew": {
        "Officers": [
            {"Name": "Sisko", "Role": "Commanding Officer"},
            {"Name": "Worf", "Role": "Strategic Operations Officer"},
        ]
    }
}`

const configStructString = `main.ShipConfig{Name:"Enterprise", ID:1701, Crew:main.Crew{Officers:[]main.Officer{main.Officer{Name:"Kirk", Role:"Commanding Officer"}, main.Officer{Name:"Spock", Role:"First Officer/Science Officer"}, main.Officer{Name:"McCoy", Role:"Chief Medical Officer"}}, Passengers:[]main.Passenger{main.Passenger{Name:"Sarek", Title:"Federation Ambassador"}}}, Active:true}`

var _ = Describe("ServiceConfig", func() {
	var runTestServiceWithExitCode = func(command *exec.Cmd, exitCode int) (stdout, stderr string) {
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		session.Wait(30 * time.Second)
		Expect(session).To(gexec.Exit(exitCode))

		return string(session.Out.Contents()), string(session.Err.Contents())
	}

	var runTestService = func(command *exec.Cmd) (stdout, stderr string) {
		return runTestServiceWithExitCode(command, 0)
	}

	var writeFile = func(fileName, contents string) (filePath string) {
		filePath = filepath.Join(tempDir, "flag-config.json")

		err := ioutil.WriteFile(filePath, []byte(configJSON), os.ModePerm)
		Expect(err).ToNot(HaveOccurred())

		return filePath
	}

	var whitespacePattern = regexp.MustCompile("\\s+")
	var command *exec.Cmd

	Context("When a config flag is passed", func() {
		It("Reads the config, from the flag JSON string", func() {
			command = exec.Command(
				binaryPath,
				fmt.Sprintf("-config=%s", whitespacePattern.ReplaceAllString(configJSON, " ")),
			)

			stdout, stderr := runTestService(command)
			Expect(stdout).To(ContainSubstring("Config: %s", configStructString), "Unexpected output. STDERR:\n%s", stderr)
		})

		It("Reads the config, from the flag YAML string", func() {
			command = exec.Command(
				binaryPath,
				fmt.Sprintf("-config=%s", configYAMLOneLine),
			)

			stdout, stderr := runTestService(command)
			Expect(stdout).To(ContainSubstring("Config: %s", configStructString), "Unexpected output. STDERR:\n%s", stderr)
		})

		Context("When the CONFIG env var is ALSO set", func() {
			It("Reads the config, from the flag string", func() {
				configString := whitespacePattern.ReplaceAllString(configJSON, " ")
				configStringAlt := whitespacePattern.ReplaceAllString(configJSONAlt, " ")

				command = exec.Command(
					binaryPath,
					fmt.Sprintf("-config=%s", configString),
				)
				command.Env = []string{
					fmt.Sprintf("CONFIG=%s", configStringAlt),
				}

				stdout, stderr := runTestService(command)
				Expect(stdout).To(ContainSubstring("Config: %s", configStructString), "Unexpected output. STDERR:\n%s", stderr)
			})
		})
	})

	Context("When a configPath flag is passed", func() {
		It("Reads the config, from the JSON file path specified by flag", func() {
			configPath := writeFile("flag-config.json", configJSON)

			command = exec.Command(
				binaryPath,
				fmt.Sprintf("-configPath=%s", configPath),
			)

			stdout, stderr := runTestService(command)
			Expect(stdout).To(ContainSubstring("Config: %s", configStructString), "Unexpected output. STDERR:\n%s", stderr)
		})

		It("Reads the config, from the YAML file path specified by flag", func() {
			configPath := writeFile("flag-config.yaml", configYAML)

			command = exec.Command(
				binaryPath,
				fmt.Sprintf("-configPath=%s", configPath),
			)

			stdout, stderr := runTestService(command)
			Expect(stdout).To(ContainSubstring("Config: %s", configStructString), "Unexpected output. STDERR:\n%s", stderr)
		})

		Context("When a CONFIG_PATH env var is ALSO set", func() {
			It("Reads the config, from the file path specified by flag", func() {
				configPath := writeFile("flag-config.json", configJSON)
				configPathAlt := writeFile("flag-env-var-config.json", configJSONAlt)

				command = exec.Command(
					binaryPath,
					fmt.Sprintf("-configPath=%s", configPath),
				)
				command.Env = []string{
					fmt.Sprintf("CONFIG_PATH=%s", configPathAlt),
				}

				stdout, stderr := runTestService(command)
				Expect(stdout).To(ContainSubstring("Config: %s", configStructString), "Unexpected output. STDERR:\n%s", stderr)
			})
		})
	})

	Context("When a CONFIG env var is set", func() {
		It("Reads the config, from the env var JSON string", func() {
			command = exec.Command(binaryPath)
			command.Env = []string{
				fmt.Sprintf("CONFIG=%s", whitespacePattern.ReplaceAllString(configJSON, " ")),
			}

			stdout, stderr := runTestService(command)
			Expect(stdout).To(ContainSubstring("Config: %s", configStructString), "Unexpected output. STDERR:\n%s", stderr)
		})

		It("Reads the config, from the env var YAML string", func() {
			command = exec.Command(binaryPath)
			command.Env = []string{
				fmt.Sprintf("CONFIG=%s", configYAMLOneLine),
			}

			stdout, stderr := runTestService(command)
			Expect(stdout).To(ContainSubstring("Config: %s", configStructString), "Unexpected output. STDERR:\n%s", stderr)
		})
	})

	Context("When a CONFIG_PATH env var is set", func() {
		It("Reads the config, from the JSON file path specified by env var", func() {
			configPath := writeFile("env-var-config.json", configJSON)

			command := exec.Command(binaryPath)
			command.Env = []string{
				fmt.Sprintf("CONFIG_PATH=%s", configPath),
			}

			stdout, stderr := runTestService(command)
			Expect(stdout).To(ContainSubstring("Config: %s", configStructString), "Unexpected output. STDERR:\n%s", stderr)
		})

		It("Reads the config, from the YAML file path specified by env var", func() {
			configPath := writeFile("env-var-config.yaml", configYAML)

			command := exec.Command(binaryPath)
			command.Env = []string{
				fmt.Sprintf("CONFIG_PATH=%s", configPath),
			}

			stdout, stderr := runTestService(command)
			Expect(stdout).To(ContainSubstring("Config: %s", configStructString), "Unexpected output. STDERR:\n%s", stderr)
		})
	})

	Context("When the help flag is passed", func() {

		var (
			stdout, stderr string
		)

		BeforeEach(func() {
			command = exec.Command(
				binaryPath,
				"-h",
			)

			helpExitCode := 2
			stdout, stderr = runTestServiceWithExitCode(command, helpExitCode)
		})

		It("Prints the list of config flags", func() {
			Expect(stderr).To(ContainSubstring("-config"))
			Expect(stderr).To(ContainSubstring("-configPath"))
		})

		It("Prints the default config options", func() {
			Expect(stderr).To(ContainSubstring("Default config values:"))
			Expect(stderr).To(MatchRegexp("Active:\\s*true"))
		})
	})
})
