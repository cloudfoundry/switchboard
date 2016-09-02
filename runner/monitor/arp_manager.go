package monitor

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/pivotal-golang/lager"
)

//go:generate counterfeiter . ArpManager
type ArpManager interface {
	ClearCache(ip string) error
	IsCached(ip string) bool
}

type ArpManagerCmd struct {
	logger lager.Logger
}

func NewArmManager(logger lager.Logger) ArpManager {
	return &ArpManagerCmd{
		logger: logger,
	}
}

func (a ArpManagerCmd) ClearCache(ip string) error {

	output, err := exec.Command("arp", "-d", ip).CombinedOutput()

	if err != nil {
		return errors.New(fmt.Sprintf("failed to delete arp entry: OUTPUT=%s, ERROR=%s", output, err.Error()))
	} else {
		return nil
	}
}

func (a ArpManagerCmd) IsCached(ip string) bool {

	output, err := exec.Command("arp", ip).CombinedOutput()
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
