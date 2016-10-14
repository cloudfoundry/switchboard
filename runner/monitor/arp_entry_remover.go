package monitor

import (
	"errors"
	"fmt"
	"net"
	"os/exec"
)

const (
	FlushARPBinPath = "/var/vcap/packages/switchboard/bin/flusharp"
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

type ARPFlusher struct {
	runner CmdRunner
}

func NewARPFlusher(runner CmdRunner) ArpEntryRemover {
	return &ARPFlusher{
		runner: runner,
	}
}

func (a ARPFlusher) RemoveEntry(ip net.IP) error {
	if ip == nil {
		return errors.New("failed to delete arp entry: invalid IP")
	}

	output, err := a.runner.Run(FlushARPBinPath, ip.String())

	if err != nil {
		return errors.New(fmt.Sprintf("failed to delete arp entry: OUTPUT=%s, ERROR=%s", output, err.Error()))
	} else {
		return nil
	}
}
