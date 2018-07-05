package healthcheck

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"net/http"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-incubator/galera-healthcheck/config"
)

const (
	STATE_JOINING        = 1
	STATE_DONOR_DESYNCED = 2
	STATE_JOINED         = 3
	STATE_SYNCED         = 4
)

type HealthChecker struct {
	db     *sql.DB
	config config.Config
	logger lager.Logger
}

func New(db *sql.DB, config config.Config, logger lager.Logger) *HealthChecker {
	return &HealthChecker{
		db:     db,
		config: config,
		logger: logger,
	}
}

func (h *HealthChecker) CheckReq(req *http.Request) (string, error) {
	return h.Check()
}

func (h *HealthChecker) Check() (string, error) {
	if h.config.Monit.ServiceName == "garbd" {
		return "", errors.New("arbitrator node")
	}

	var unused string
	var value int
	err := h.db.QueryRow("SHOW STATUS LIKE 'wsrep_local_state'").Scan(&unused, &value)

	if err == sql.ErrNoRows {
		return "", errors.New("wsrep_local_state variable not set (possibly not a galera db)")
	} else if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			return "", errors.New("Cannot get status from galera")
		} else {
			return "", err
		}
	}

	switch value {
	case STATE_JOINING:
		return "", errors.New("joining")
	case STATE_DONOR_DESYNCED:
		if h.config.AvailableWhenDonor {
			return h.healthy(value)
		}
		return "", errors.New("not synced")
	case STATE_JOINED:
		return "", errors.New("joined")
	case STATE_SYNCED:
		return h.healthy(value)
	default:
		return "", fmt.Errorf("Unrecognized state: %d", value)
	}

}

func (h *HealthChecker) healthy(value int) (string, error) {
	if !h.config.AvailableWhenReadOnly {
		readOnly, err := h.isReadOnly()
		if err != nil {
			return "", err
		}

		if readOnly {
			return "", errors.New("read-only")
		}
	}
	return "synced", nil
}

func (h *HealthChecker) isReadOnly() (bool, error) {
	var unused, readOnly string
	err := h.db.QueryRow("SHOW GLOBAL VARIABLES LIKE 'read_only'").Scan(&unused, &readOnly)
	if err != nil {
		return false, err
	}

	if readOnly == "ON" {
		return true, nil
	}
	return false, nil
}
