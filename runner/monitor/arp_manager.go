package monitor

import (
	"errors"
	"fmt"
	"os/exec"

	"code.cloudfoundry.org/lager"
)

//go:generate counterfeiter . ArpManager
type ArpManager interface {
	ClearCache(ip string) error
	IsCached(ip string) bool
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

func NewArmManager(runner CmdRunner, logger lager.Logger) ArpManager {
	return &ArpManagerCmd{
		runner: runner,
		logger: logger,
	}
}

func (a ArpManagerCmd) ClearCache(ip string) error {
	output, err := a.runner.Run("arp", "-d", ip)

	if err != nil {
		return errors.New(fmt.Sprintf("failed to delete arp entry: OUTPUT=%s, ERROR=%s", output, err.Error()))
	} else {
		return nil
	}
}

func (a ArpManagerCmd) IsCached(ip string) bool {
	output, err := a.runner.Run("arp", ip)
	if err != nil {
		a.logger.Info(fmt.Sprintf("arp didnt find %s in cache, skipping cache invalidation", ip), lager.Data{
			"err":    err.Error(),
			"output": output,
		})
		return false
	} else {
		a.logger.Info(fmt.Sprintf("arp found %s in cache", ip))
		return true
	}
}
