package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-incubator/galera-healthcheck/api"
	"github.com/cloudfoundry-incubator/galera-healthcheck/config"
	"github.com/cloudfoundry-incubator/galera-healthcheck/healthcheck"
	"github.com/cloudfoundry-incubator/galera-healthcheck/mysqld_cmd"
	"github.com/cloudfoundry-incubator/galera-healthcheck/sequence_number"

	"github.com/cloudfoundry-incubator/galera-healthcheck/monit_client"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	rootConfig, err := config.NewConfig(os.Args)

	logger := rootConfig.Logger

	err = rootConfig.Validate()
	if err != nil {
		logger.Fatal("Failed to validate config", err)
	}

	db, err := sql.Open("mysql",
		fmt.Sprintf("%s:%s@tcp(%s:%d)/",
			rootConfig.DB.User,
			rootConfig.DB.Password,
			rootConfig.DB.Host,
			rootConfig.DB.Port))

	if err != nil {
		logger.Error("Failed to open DB connection", err, lager.Data{
			"dbHost": rootConfig.DB.Host,
			"dbPort": rootConfig.DB.Port,
			"dbUser": rootConfig.DB.User,
		})
	} else {
		logger.Info("Opened DB connection", lager.Data{
			"dbHost": rootConfig.DB.Host,
			"dbPort": rootConfig.DB.Port,
			"dbUser": rootConfig.DB.User,
		})
	}

	mysqldCmd := mysqld_cmd.NewMysqldCmd(logger, *rootConfig)
	monitClient := monit_client.New(rootConfig.Monit, logger)
	healthchecker := healthcheck.New(db, *rootConfig, logger)
	sequenceNumberchecker := sequence_number.New(db, mysqldCmd, *rootConfig, logger)
	stateSnapshotter := &healthcheck.DBStateSnapshotter{
		DB:     db,
		Logger: logger,
	}

	router, err := api.NewRouter(
		logger,
		rootConfig,
		monitClient,
		sequenceNumberchecker,
		healthchecker,
		healthchecker,
		stateSnapshotter,
	)
	if err != nil {
		logger.Fatal("Failed to create router", err)
	}

	address := fmt.Sprintf("%s:%d", rootConfig.Host, rootConfig.Port)
	url := fmt.Sprintf("http://%s/", address)
	logger.Info("Serving healthcheck endpoint", lager.Data{
		"url": url,
	})

	go func() {
		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		logger.Info("Attempting to GET endpoint...", lager.Data{
			"url": url,
		})

		var resp *http.Response
		retryAttemptsRemaining := 3
		for ; retryAttemptsRemaining > 0; retryAttemptsRemaining-- {
			resp, err = client.Get(url)
			if err != nil {
				logger.Info("GET endpoint failed, retrying...", lager.Data{
					"url": url,
					"err": err,
				})
				time.Sleep(time.Second * 10)
			} else {
				break
			}
		}
		if retryAttemptsRemaining == 0 {
			logger.Fatal(
				"Initialization failed: Coudn't GET endpoint",
				err,
				lager.Data{
					"url":     url,
					"retries": retryAttemptsRemaining,
				})
		}
		logger.Info("GET endpoint succeeded, now accepting connections", lager.Data{
			"url":        url,
			"statusCode": resp.StatusCode,
		})

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logger.Fatal("Initialization failed: reading response body", err, lager.Data{
				"url":         url,
				"status-code": resp.StatusCode,
			})
		}
		logger.Info(fmt.Sprintf("Initial Response: %s", body))

		// existence of pid file means the server is running
		pid := os.Getpid()
		err = ioutil.WriteFile(rootConfig.PidFile, []byte(strconv.Itoa(os.Getpid())), 0644)
		if err != nil {
			logger.Fatal("Failed to write pid file", err, lager.Data{
				"pid":     pid,
				"pidFile": rootConfig.PidFile,
			})
		}

		// Used by tests to deterministically know that the healthcheck is accepting incoming connections
		logger.Info("Healthcheck Started")
	}()

	err = http.ListenAndServe(address, router)
	if err != nil {
		logger.Fatal("Galera healthcheck stopped unexpectedly", err)
	}
	logger.Info("Galera healthcheck has stopped gracefully")
}
