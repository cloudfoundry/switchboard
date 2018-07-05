package sequence_number

import (
	"database/sql"
	"errors"
	"fmt"

	"strconv"

	"net/http"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-incubator/galera-healthcheck/config"
	"github.com/cloudfoundry-incubator/galera-healthcheck/mysqld_cmd"
)

type SequenceNumberChecker struct {
	db        *sql.DB
	config    config.Config
	logger    lager.Logger
	mysqldCmd mysqld_cmd.MysqldCmd
}

func New(db *sql.DB,
	mysqldCmd mysqld_cmd.MysqldCmd,
	config config.Config,
	logger lager.Logger,
) *SequenceNumberChecker {
	return &SequenceNumberChecker{
		db:        db,
		config:    config,
		logger:    logger,
		mysqldCmd: mysqldCmd,
	}
}

func (s *SequenceNumberChecker) Check(req *http.Request) (string, error) {
	s.logger.Info("Checking sequence number of mariadb node...")

	if s.config.Monit.ServiceName == "garbd" {
		return "no sequence number - running on arbitrator node", nil
	} else if s.dbReachable() {
		return "", errors.New("can't determine sequence number when database is running")
	} else {
		returnedSeqNo, err := s.readSeqNoFromRecoverCmd()
		if err != nil {
			return "", err
		}

		returnedSeqNoInt, converr := strconv.Atoi(returnedSeqNo)
		if converr != nil {
			return "", converr
		}

		if returnedSeqNoInt < 0 {
			return "", errors.New(fmt.Sprintf("Invalid sequence number %s", returnedSeqNo))
		}

		return returnedSeqNo, nil
	}
}

func (s *SequenceNumberChecker) readSeqNoFromRecoverCmd() (string, error) {
	s.logger.Info("Reading seqno from logs")
	seqno, err := s.mysqldCmd.RecoverSeqno()
	if err != nil {
		s.logger.Error("Failed to retrieve seqno from logs", err)
		return "", err
	}

	return seqno, nil
}

func (s *SequenceNumberChecker) dbReachable() bool {
	_, err := s.db.Exec("SHOW VARIABLES")
	if err != nil {
		s.logger.Info(fmt.Sprintf("Database not reachable, continuing: %s", err.Error()))
	}
	return err == nil
}
