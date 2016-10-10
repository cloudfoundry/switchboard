package locket

import (
	"errors"
	"os"
	"strings"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/consuladapter"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/runtimeschema/metric"
	"github.com/nu7hatch/gouuid"
)

var (
	ErrLockLost = errors.New("lock lost")
)

type Lock struct {
	consul *Session
	key    string
	value  []byte

	clock         clock.Clock
	retryInterval time.Duration

	logger lager.Logger

	lockAcquiredMetric metric.Metric
	lockUptimeMetric   metric.Duration
	lockAcquiredTime   time.Time
}

func NewLock(
	logger lager.Logger,
	consulClient consuladapter.Client,
	lockKey string,
	lockValue []byte,
	clock clock.Clock,
	retryInterval time.Duration,
	lockTTL time.Duration,
) Lock {
	lockMetricName := strings.Replace(lockKey, "/", "-", -1)

	uuid, err := uuid.NewV4()
	if err != nil {
		logger.Fatal("create-uuid-failed", err)
	}

	session, err := NewSessionNoChecks(uuid.String(), lockTTL, consulClient)
	if err != nil {
		logger.Fatal("consul-session-failed", err)
	}

	return Lock{
		consul: session,
		key:    lockKey,
		value:  lockValue,

		clock:         clock,
		retryInterval: retryInterval,

		logger: logger,

		lockAcquiredMetric: metric.Metric("LockHeld." + lockMetricName),
		lockUptimeMetric:   metric.Duration("LockHeldDuration." + lockMetricName),
	}
}

func (l Lock) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	logger := l.logger.Session("lock", lager.Data{"key": l.key, "value": string(l.value)})
	logger.Info("starting")

	defer func() {
		l.consul.Destroy()
		logger.Info("done")
	}()

	acquireErr := make(chan error, 1)

	acquire := func(session *Session) {
		logger.Info("acquiring-lock")
		acquireErr <- session.AcquireLock(l.key, l.value)
	}

	var c <-chan time.Time
	var reemit <-chan time.Time

	go acquire(l.consul)

	for {
		select {
		case sig := <-signals:
			logger.Info("shutting-down", lager.Data{"received-signal": sig})

			logger.Debug("releasing-lock")
			l.consul.Destroy()
			l.emitMetrics(false)
			return nil
		case err := <-l.consul.Err():
			if ready == nil {
				logger.Error("lost-lock", err)
				l.emitMetrics(false)
				return ErrLockLost
			}

			logger.Error("consul-error-without-lock", err)
		case err := <-acquireErr:
			if err != nil {
				logger.Error("acquire-lock-failed", err)
				l.emitMetrics(false)
				c = l.clock.NewTimer(l.retryInterval).C()
				break
			}

			logger.Info("acquire-lock-succeeded")
			l.lockAcquiredTime = l.clock.Now()
			l.emitMetrics(true)
			reemit = l.clock.NewTimer(30 * time.Second).C()
			close(ready)
			ready = nil
			c = nil
			logger.Info("started")
		case <-reemit:
			l.emitMetrics(true)
			reemit = l.clock.NewTimer(30 * time.Second).C()
		case <-c:
			logger.Info("retrying-acquiring-lock")
			newSession, err := l.consul.Recreate()
			if err != nil {
				logger.Error("failed-to-recreate-session", err)
				c = l.clock.NewTimer(l.retryInterval).C()
			} else {
				l.consul = newSession
				c = nil
				go acquire(newSession)
			}
		}
	}
}

func (l Lock) emitMetrics(acquired bool) {
	var acqVal int
	var uptime time.Duration

	if acquired {
		acqVal = 1
		uptime = l.clock.Since(l.lockAcquiredTime)
	} else {
		acqVal = 0
		uptime = 0
	}

	l.logger.Debug("reemit-lock-uptime", lager.Data{"uptime": uptime,
		"uptimeMetricName":       l.lockUptimeMetric,
		"lockAcquiredMetricName": l.lockAcquiredMetric,
	})
	err := l.lockUptimeMetric.Send(uptime)
	if err != nil {
		l.logger.Error("failed-to-send-lock-uptime-metric", err)
	}

	err = l.lockAcquiredMetric.Send(acqVal)
	if err != nil {
		l.logger.Error("failed-to-send-lock-acquired-metric", err)
	}
}
