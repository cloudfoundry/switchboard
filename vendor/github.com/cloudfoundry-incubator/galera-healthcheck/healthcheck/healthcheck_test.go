package healthcheck_test

import (
	"errors"
	"fmt"

	"database/sql"

	testdb "github.com/erikstmartin/go-testdb"

	"code.cloudfoundry.org/lager/lagertest"
	"github.com/cloudfoundry-incubator/galera-healthcheck/config"
	"github.com/cloudfoundry-incubator/galera-healthcheck/healthcheck"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("GaleraHealthChecker", func() {
	Describe("Check", func() {
		Context("Node is running mysql", func() {

			Context("when WSREP_STATUS is joining", func() {
				It("returns false and joining", func() {
					config := healthcheckTestHelperConfig{
						wsrepStatus: healthcheck.STATE_JOINING,
						monit: config.MonitConfig{
							ServiceName: "mariadb_ctrl",
						},
					}

					_, err := healthcheckTestHelper(config)
					Expect(err).To(MatchError("joining"))
				})
			})

			Context("when WSREP_STATUS is joined", func() {
				It("returns false and joined", func() {
					config := healthcheckTestHelperConfig{
						wsrepStatus: healthcheck.STATE_JOINED,
						monit: config.MonitConfig{
							ServiceName: "mariadb_ctrl",
						},
					}
					_, err := healthcheckTestHelper(config)
					Expect(err).To(MatchError("joined"))

				})
			})

			Context("when WSREP_STATUS is donor", func() {
				Context("when not AVAILABLE_WHEN_DONOR", func() {
					It("returns false and not-synced", func() {
						config := healthcheckTestHelperConfig{
							wsrepStatus: healthcheck.STATE_DONOR_DESYNCED,
							monit: config.MonitConfig{
								ServiceName: "mariadb_ctrl",
							},
						}
						_, err := healthcheckTestHelper(config)
						Expect(err).To(MatchError("not synced"))

					})
				})

				Context("when AVAILABLE_WHEN_DONOR", func() {
					Context("when READ_ONLY is ON", func() {
						Context("when AVAILABLE_WHEN_READONLY is true", func() {
							It("returns true and synced", func() {
								config := healthcheckTestHelperConfig{
									wsrepStatus:           healthcheck.STATE_DONOR_DESYNCED,
									readOnly:              true,
									availableWhenDonor:    true,
									availableWhenReadOnly: true,
								}
								result, err := healthcheckTestHelper(config)
								Expect(err).ToNot(HaveOccurred())
								Expect(result).To(Equal("synced"))
							})
						})

						Context("when AVAILABLE_WHEN_READONLY is false", func() {
							It("returns false and read-only", func() {
								config := healthcheckTestHelperConfig{
									wsrepStatus:        healthcheck.STATE_DONOR_DESYNCED,
									readOnly:           true,
									availableWhenDonor: true,
								}
								_, err := healthcheckTestHelper(config)
								Expect(err).To(MatchError("read-only"))
							})
						})
					})

					Context("when READ_ONLY is OFF", func() {
						It("returns true and synced", func() {
							config := healthcheckTestHelperConfig{
								wsrepStatus:        healthcheck.STATE_DONOR_DESYNCED,
								availableWhenDonor: true,
							}

							result, err := healthcheckTestHelper(config)
							Expect(err).ToNot(HaveOccurred())
							Expect(result).To(Equal("synced"))
						})
					})
				})

			})

			Context("when WSREP_STATUS is synced", func() {
				Context("when READ_ONLY is ON", func() {
					Context("when AVAILABLE_WHEN_READONLY is true", func() {
						It("returns true and synced", func() {
							config := healthcheckTestHelperConfig{
								wsrepStatus:           healthcheck.STATE_SYNCED,
								readOnly:              true,
								availableWhenReadOnly: true,
								monit: config.MonitConfig{
									ServiceName: "mariadb_ctrl",
								},
							}

							result, err := healthcheckTestHelper(config)
							Expect(err).ToNot(HaveOccurred())
							Expect(result).To(Equal("synced"))
						})
					})

					Context("when AVAILABLE_WHEN_READONLY is false", func() {
						It("returns false and read-only", func() {
							config := healthcheckTestHelperConfig{
								wsrepStatus: healthcheck.STATE_SYNCED,
								readOnly:    true,
								monit: config.MonitConfig{
									ServiceName: "mariadb_ctrl",
								},
							}
							_, err := healthcheckTestHelper(config)
							Expect(err).To(MatchError("read-only"))
						})
					})
				})

				Context("when READ_ONLY is OFF", func() {
					It("returns true and synced", func() {
						config := healthcheckTestHelperConfig{
							wsrepStatus: healthcheck.STATE_SYNCED,
							monit: config.MonitConfig{
								ServiceName: "mariadb_ctrl",
							},
						}

						result, err := healthcheckTestHelper(config)
						Expect(err).ToNot(HaveOccurred())
						Expect(result).To(Equal("synced"))
					})
				})
			})

			Context("when SHOW STATUS returns an error", func() {
				It("returns false and the error message", func() {
					db, _ := sql.Open("testdb", "")

					sql := "SHOW STATUS LIKE 'wsrep_local_state'"
					testdb.StubQueryError(sql, errors.New("test error"))

					config := config.Config{
						AvailableWhenDonor:    false,
						AvailableWhenReadOnly: false,
						Monit: config.MonitConfig{
							ServiceName: "mariadb_ctrl",
						},
					}

					logger := lagertest.NewTestLogger("healthcheck test")
					healthchecker := healthcheck.New(db, config, logger)

					_, err := healthchecker.Check()
					Expect(err).To(MatchError("test error"))
				})
			})

			Context("when SHOW GLOBAL VARIABLES LIKE returns an error", func() {
				It("returns false and the error message", func() {
					db, _ := sql.Open("testdb", "")

					sql := "SHOW STATUS LIKE 'wsrep_local_state'"
					columns := []string{"Variable_name", "Value"}
					result := "wsrep_local_state,4"
					testdb.StubQuery(sql, testdb.RowsFromCSVString(columns, result))

					sql = "SHOW GLOBAL VARIABLES LIKE 'read_only'"
					testdb.StubQueryError(sql, errors.New("another test error"))

					config := config.Config{
						AvailableWhenDonor:    false,
						AvailableWhenReadOnly: false,
						Monit: config.MonitConfig{
							ServiceName: "mariadb_ctrl",
						},
					}

					logger := lagertest.NewTestLogger("healthcheck test")
					healthchecker := healthcheck.New(db, config, logger)

					_, err := healthchecker.Check()
					Expect(err).To(MatchError("another test error"))
				})
			})

			Context("db is down", func() {
				var healthchecker *healthcheck.HealthChecker

				BeforeEach(func() {
					db, _ := sql.Open("testdb", "")

					config := config.Config{
						AvailableWhenDonor:    false,
						AvailableWhenReadOnly: false,
						Monit: config.MonitConfig{
							ServiceName: "mariadb_ctrl",
						},
					}

					err := fmt.Errorf("connection refused")
					testdb.StubQueryError("SHOW STATUS LIKE 'wsrep_local_state'", err)

					logger := lagertest.NewTestLogger("healthcheck test")
					healthchecker = healthcheck.New(db, config, logger)
				})

				It("returns false and a warning message", func() {
					_, err := healthchecker.Check()
					Expect(err).To(MatchError("Cannot get status from galera"))
				})

			})
		})

		Context("Node is running garbd", func() {

			It("returns true and a message indicating that this is arbitrator node", func() {
				config := healthcheckTestHelperConfig{
					monit: config.MonitConfig{
						ServiceName: "garbd",
					},
				}

				_, err := healthcheckTestHelper(config)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("arbitrator node"))
			})
		})
	})
})

type healthcheckTestHelperConfig struct {
	wsrepStatus           int
	readOnly              bool
	availableWhenDonor    bool
	availableWhenReadOnly bool
	monit                 config.MonitConfig
}

func healthcheckTestHelper(testConfig healthcheckTestHelperConfig) (string, error) {
	db, _ := sql.Open("testdb", "")

	sql := "SHOW STATUS LIKE 'wsrep_local_state'"
	columns := []string{"Variable_name", "Value"}
	result := fmt.Sprintf("wsrep_local_state,%d", testConfig.wsrepStatus)
	testdb.StubQuery(sql, testdb.RowsFromCSVString(columns, result))

	sql = "SHOW GLOBAL VARIABLES LIKE 'read_only'"
	columns = []string{"Variable_name", "Value"}
	var readOnlyText string
	if testConfig.readOnly {
		readOnlyText = "ON"
	} else {
		readOnlyText = "OFF"
	}
	result = fmt.Sprintf("read_only,%s", readOnlyText)
	testdb.StubQuery(sql, testdb.RowsFromCSVString(columns, result))

	config := config.Config{
		AvailableWhenDonor:    testConfig.availableWhenDonor,
		AvailableWhenReadOnly: testConfig.availableWhenReadOnly,
		Monit: testConfig.monit,
	}

	logger := lagertest.NewTestLogger("healthcheck test")
	healthchecker := healthcheck.New(db, config, logger)

	return healthchecker.Check()
}
