package mysqld_cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-incubator/galera-healthcheck/config"
)

type MysqldCmd interface {
	RecoverSeqno() (string, error)
}

type mysqldCmd struct {
	logger       lager.Logger
	mysqldconfig config.Config
}

func NewMysqldCmd(logger lager.Logger, mysqldconfig config.Config) MysqldCmd {
	return &mysqldCmd{
		logger:       logger,
		mysqldconfig: mysqldconfig,
	}
}

/*
* Why?
*
* Galera does not provide an elegant way to determine seqno if the DB is not
* running.
* The mysqld --wsrep-recover cmd prints the seqno to stderr (lines starts with `WSREP: Recovered position:`)
* This command writes its stderr to a log file specified by the `--log-error`
* flag
 */
func (m *mysqldCmd) RecoverSeqno() (string, error) {

	errorLogFile := path.Join(os.TempDir(), "galera-healthcheck-mysqld-log.err")
	os.RemoveAll(errorLogFile) //ensure log is empty

	cmd := exec.Command(m.mysqldconfig.MysqldPath,
		fmt.Sprintf("--defaults-file=%s", m.mysqldconfig.MyCnfPath),
		"--wsrep-recover",
		fmt.Sprintf("--log-error=%s", errorLogFile))

	stdout, cmdErr := cmd.CombinedOutput()
	stderr, readingLogErr := ioutil.ReadFile(errorLogFile)
	if readingLogErr != nil {
		stderr = []byte("failed to read stderr")
	}

	if cmdErr != nil {
		m.logger.Error("Error running mysqld recovery", cmdErr, lager.Data{
			"stdout": stdout,
			"stderr": stderr,
		})
		return "", cmdErr
	} else {
		m.logger.Debug(string(stdout))
	}

	seqNoRegex := `WSREP: Recovered position:.*:(\d+)`
	re := regexp.MustCompile(seqNoRegex)
	sequenceNumberLogLine := re.FindStringSubmatch(string(stderr))

	if len(sequenceNumberLogLine) < 2 {
		// First match is the whole string, second match is the seq no
		err := errors.New(fmt.Sprintf("Couldn't find regex: %s Log Line: %s", seqNoRegex, sequenceNumberLogLine))
		m.logger.Error("Failed to parse seqno from logs", err)
		return "", err
	}

	sequenceNumber := sequenceNumberLogLine[1]
	return sequenceNumber, nil
}
