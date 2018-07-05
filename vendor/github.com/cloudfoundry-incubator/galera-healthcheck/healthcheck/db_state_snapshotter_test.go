package healthcheck_test

import (
	"github.com/cloudfoundry-incubator/galera-healthcheck/domain"
	. "github.com/cloudfoundry-incubator/galera-healthcheck/healthcheck"

	"database/sql"
	"errors"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DBStateSnapshotter", func() {
	Describe("State", func() {
		var (
			snapshotter *DBStateSnapshotter
			db          *sql.DB
			err         error
			mock        sqlmock.Sqlmock

			logger lager.Logger
		)

		BeforeEach(func() {
			db, mock, err = sqlmock.New()
			Expect(err).NotTo(HaveOccurred())

			logger = lagertest.NewTestLogger("snapshotter")

			snapshotter = &DBStateSnapshotter{
				DB:     db,
				Logger: logger,
			}
		})

		AfterEach(func() {
			Expect(mock.ExpectationsWereMet()).To(Succeed())
		})

		It("queries for the 'wsrep_local_state', 'wsrep_local_index', and 'read_only' attributes in a transaction", func() {
			mock.ExpectBegin()
			mock.ExpectQuery("SHOW STATUS LIKE 'wsrep_local_state'").WillReturnRows(sqlmock.NewRows([]string{"Variable_Name", "Value"}).AddRow("wsrep_local_state", 4))
			mock.ExpectQuery("SHOW STATUS LIKE 'wsrep_local_index'").WillReturnRows(sqlmock.NewRows([]string{"Variable_Name", "Value"}).AddRow("wsrep_local_index", 0))
			mock.ExpectQuery("SHOW GLOBAL VARIABLES LIKE 'read_only'").WillReturnRows(sqlmock.NewRows([]string{"Variable_Name", "Value"}).AddRow("read_only", "ON"))
			mock.ExpectCommit()

			state, err := snapshotter.State()
			Expect(err).NotTo(HaveOccurred())

			Expect(state.WsrepLocalIndex).To(Equal(uint(0)))
			Expect(state.WsrepLocalState).To(Equal(domain.Synced))
			Expect(state.ReadOnly).To(BeTrue())
		})

		It("returns an error when it can't make a transaction", func() {
			mock.ExpectBegin().WillReturnError(errors.New("error"))

			_, err = snapshotter.State()

			Expect(err).To(MatchError(errors.New("error")))
		})

		It("returns an error and rolls back when it can't query 'wsrep_local_state'", func() {
			mock.ExpectBegin()
			mock.ExpectQuery("SHOW STATUS LIKE 'wsrep_local_state'").WillReturnError(errors.New("error"))
			mock.ExpectRollback()

			_, err = snapshotter.State()

			Expect(err).To(MatchError(errors.New("error")))
		})

		It("returns an error and rolls back when it can't query 'wsrep_local_index'", func() {
			mock.ExpectBegin()
			mock.ExpectQuery("SHOW STATUS LIKE 'wsrep_local_state'").WillReturnRows(sqlmock.NewRows([]string{"Variable_Name", "Value"}).AddRow("wsrep_local_state", 4))
			mock.ExpectQuery("SHOW STATUS LIKE 'wsrep_local_index'").WillReturnError(errors.New("error"))
			mock.ExpectRollback()

			_, err = snapshotter.State()

			Expect(err).To(MatchError(errors.New("error")))
		})

		It("returns an error and rolls back when it can't query 'read_only'", func() {
			mock.ExpectBegin()
			mock.ExpectQuery("SHOW STATUS LIKE 'wsrep_local_state'").WillReturnRows(sqlmock.NewRows([]string{"Variable_Name", "Value"}).AddRow("wsrep_local_state", 4))
			mock.ExpectQuery("SHOW STATUS LIKE 'wsrep_local_index'").WillReturnRows(sqlmock.NewRows([]string{"Variable_Name", "Value"}).AddRow("wsrep_local_index", 0))
			mock.ExpectQuery("SHOW GLOBAL VARIABLES LIKE 'read_only'").WillReturnError(errors.New("error"))
			mock.ExpectRollback()

			_, err = snapshotter.State()

			Expect(err).To(MatchError(errors.New("error")))
		})

		It("returns an error when it can't commit the transaction", func() {
			mock.ExpectBegin()
			mock.ExpectQuery("SHOW STATUS LIKE 'wsrep_local_state'").WillReturnRows(sqlmock.NewRows([]string{"Variable_Name", "Value"}).AddRow("wsrep_local_state", 4))
			mock.ExpectQuery("SHOW STATUS LIKE 'wsrep_local_index'").WillReturnRows(sqlmock.NewRows([]string{"Variable_Name", "Value"}).AddRow("wsrep_local_index", 0))
			mock.ExpectQuery("SHOW GLOBAL VARIABLES LIKE 'read_only'").WillReturnRows(sqlmock.NewRows([]string{"Variable_Name", "Value"}).AddRow("read_only", "ON"))
			mock.ExpectCommit().WillReturnError(errors.New("error"))

			_, err = snapshotter.State()

			Expect(err).To(MatchError(errors.New("error")))
		})
	})
})
