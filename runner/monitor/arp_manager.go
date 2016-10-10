package monitor

import (
	"errors"
	"fmt"
	"os/exec"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter . ArpManager
type ArpManager interface {
	RemoveEntry(ip string) error
}

//go:generate counterfeiter . CmdRunner
type CmdRunner interface {
	Run(name string, cmd ...string) ([]byte, error)
}

type ExecCmdRunner struct{}

func (r *ExecCmdRunner) Run(name string, arg ...string) ([]byte, error) {
	return exec.Command(name, arg...).CombinedOutput()
}

type ArpManagerCmd struct {
	runner CmdRunner
	logger lager.Logger
}

func NewArpManager(runner CmdRunner, logger lager.Logger) ArpManager {
	return &ArpManagerCmd{
		runner: runner,
		logger: logger,
	}
}

func (a ArpManagerCmd) RemoveEntry(ip string) error {
	output, err := a.runner.Run("/usr/sbin/arp", "-d", ip)

	if err != nil {
		return errors.New(fmt.Sprintf("failed to delete arp entry: OUTPUT=%s, ERROR=%s", output, err.Error()))
	} else {
		return nil
	}
}
