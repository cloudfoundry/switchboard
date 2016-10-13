package monitor

import (
	"errors"
	"fmt"
	"net"
	"os/exec"
)

//go:generate counterfeiter . ArpEntryRemover
type ArpEntryRemover interface {
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

type PrivilegedArpEntryRemover struct {
	runner CmdRunner
}

func NewPrivilegedArpEntryRemover(runner CmdRunner) ArpEntryRemover {
	return &PrivilegedArpEntryRemover{
		runner: runner,
	}
}

func (a PrivilegedArpEntryRemover) RemoveEntry(ip net.IP) error {
	if ip == nil {
		return errors.New("failed to delete arp entry: invalid IP")
	}

	output, err := a.runner.Run("/usr/sbin/arp", "-d", ip.String())

	if err != nil {
		return errors.New(fmt.Sprintf("failed to delete arp entry: OUTPUT=%s, ERROR=%s", output, err.Error()))
	} else {
		return nil
	}
}
