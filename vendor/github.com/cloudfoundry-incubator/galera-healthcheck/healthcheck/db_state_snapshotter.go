package healthcheck

import (
	"database/sql"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-incubator/galera-healthcheck/domain"
)

type DBStateSnapshotter struct {
	DB     *sql.DB
	Logger lager.Logger
}

func (s *DBStateSnapshotter) State() (state domain.DBState, err error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return domain.DBState{}, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}

		err = tx.Commit()
	}()

	var (
		unused     string
		localState domain.WsrepLocalState
		localIndex uint
		readOnly   string
	)

	err = tx.QueryRow("SHOW STATUS LIKE 'wsrep_local_state'").Scan(&unused, &localState)
	if err != nil {
		return
	}

	err = tx.QueryRow("SHOW STATUS LIKE 'wsrep_local_index'").Scan(&unused, &localIndex)
	if err != nil {
		return
	}

	err = tx.QueryRow("SHOW GLOBAL VARIABLES LIKE 'read_only'").Scan(&unused, &readOnly)
	if err != nil {
		return
	}

	return domain.DBState{
		WsrepLocalIndex: localIndex,
		WsrepLocalState: localState,
		ReadOnly:        (readOnly == "ON"),
	}, nil
}
