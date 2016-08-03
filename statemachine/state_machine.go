package statemachine

import (
	"fmt"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-incubator/switchboard/domain"
)

//go:generate stringer -type=State
type State int

const (
	Healthy State = iota
	Unhealthy
)

type StatefulStateMachine struct {
	State              State
	OnBecomesUnhealthy func(domain.Backend)
	OnBecomesHealthy   func(domain.Backend)
	Logger             lager.Logger
}

func (m *StatefulStateMachine) BecomesUnhealthy(backend domain.Backend) {
	if m.State == Healthy {
		m.Logger.Debug("StateMachine Transitioning to unhealthy", lager.Data{"backend": backend})
		m.OnBecomesUnhealthy(backend)
	}

	m.Logger.Debug("StateMachine unhealthy", lager.Data{"backend": backend})

	m.State = Unhealthy
}

func (m *StatefulStateMachine) BecomesHealthy(backend domain.Backend) {
	if m.State == Unhealthy {
		m.Logger.Debug("StateMachine Transitioning to not unhealthy", lager.Data{"backend": backend})
		m.OnBecomesHealthy(backend)
	}

	m.Logger.Debug("StateMachine not unhealthy", lager.Data{"backend": backend})

	m.State = Healthy
}

func (m *StatefulStateMachine) RemainsInSameState(backend domain.Backend) {
	m.Logger.Debug(
		"StateMachine remaining in the same state",
		lager.Data{"backend": backend, "state": fmt.Sprint(m.State)},
	)
}
