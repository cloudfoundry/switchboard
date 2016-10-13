package monitor

import (
	"errors"
	"fmt"
	"net"
	"os/exec"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter . ArpManager
type ArpManager interface {
	RemoveEntry(ip net.IP) error
}

//go:generate counterfeiter . CmdRunner
type CmdRunner interface {
	Run(name string, cmd ...string) ([]byte, error)
}

type ExecCmdRunner struct{}

func (r *ExecCmdRunner) Run(name string, arg ...string) ([]byte, error) {
	return exec.Command(name, arg...).CombinedOutput()
}

type PrivilegedArpManager struct {
	runner CmdRunner
	logger lager.Logger
}

func NewPrivilegedArpManager(runner CmdRunner, logger lager.Logger) ArpManager {
	return &PrivilegedArpManager{
		runner: runner,
		logger: logger,
	}
}

func (a PrivilegedArpManager) RemoveEntry(ip net.IP) error {
	output, err := a.runner.Run("/usr/sbin/arp", "-d", ip.String())

	if err != nil {
		return errors.New(fmt.Sprintf("failed to delete arp entry: OUTPUT=%s, ERROR=%s", output, err.Error()))
	} else {
		return nil
	}
}
