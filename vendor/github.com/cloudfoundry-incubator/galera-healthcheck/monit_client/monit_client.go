package monit_client

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-incubator/galera-healthcheck/config"
	"github.com/cloudfoundry-incubator/galera-healthcheck/mysql_start_mode"
)

type MonitClient struct {
	monitConfig config.MonitConfig
	logger      lager.Logger
}

func New(monitConfig config.MonitConfig, logger lager.Logger) *MonitClient {
	return &MonitClient{
		monitConfig: monitConfig,
		logger:      logger,
	}
}

func (m *MonitClient) GetLogger(req *http.Request) lager.Logger {
	return m.logger
}

func (m *MonitClient) StartServiceBootstrap(req *http.Request) (string, error) {
	if m.monitConfig.ServiceName == "garbd" {
		return "", errors.New("bootstrapping arbitrator not allowed")
	} else {
		return m.startService("bootstrap", true)
	}
}

func (m *MonitClient) StartServiceJoin(req *http.Request) (string, error) {
	urlParams := req.URL.Query()
	sstDisabled := !(urlParams.Get("sst") == "true")
	return m.startService("join", sstDisabled)
}

func (m *MonitClient) StartServiceSingleNode(req *http.Request) (string, error) {
	return m.startService("singleNode", true)
}

func (m *MonitClient) startService(startMode string, sstDisabled bool) (string, error) {
	if m.monitConfig.ServiceName != "garbd" {
		mySqlStartMode := mysql_start_mode.NewMysqlStartMode(m.monitConfig.MysqlStateFilePath, startMode)
		err := mySqlStartMode.Start()
		if err != nil {
			m.logger.Error("Failed to start mysql node", err)
			return "", err
		}

		if m.monitConfig.BootstrapFilePath != "" {
			prestartCmd := exec.Command(
				"/bin/bash",
				m.monitConfig.BootstrapFilePath,
			)

			env := os.Environ()
			env = append(env, fmt.Sprintf("LOG_FILE=%s", m.monitConfig.BootstrapLogFilePath))
			if sstDisabled {
				env = append(env, "DISABLE_SST=1")
				env = append(env, fmt.Sprintf("INTERRUPT_NOTIFY_CMD=%s", m.monitConfig.SstInterruptNotifyCmd))
			}
			prestartCmd.Env = env

			stdoutDest, err := os.OpenFile(m.monitConfig.BootstrapLogFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				m.logger.Error(fmt.Sprintf("Failed to open pre-start-unprivileged log file: %s", m.monitConfig.BootstrapLogFilePath), err)
				return "", err
			}
			defer stdoutDest.Close()

			prestartCmd.Stdout = stdoutDest
			prestartCmd.Stderr = stdoutDest

			err = prestartCmd.Run()
			if err != nil {
				m.logger.Error("Failed to pre-start mysql node", err)
				return "", err
			}
		}

		// Used when mariadb_ctrl does not support a bootstrap file
		if m.monitConfig.EnableSstMarkerFilePath != "" {
			if sstDisabled {
				os.Remove(m.monitConfig.EnableSstMarkerFilePath)
			} else {
				os.Create(m.monitConfig.EnableSstMarkerFilePath)
			}
		}
	}

	err := m.runServiceCmd("start")
	msg := ""
	if err == nil {
		msg = fmt.Sprintf("Successfully sent start request in %s mode", startMode)
	}
	return msg, err
}

func (m *MonitClient) StopService(req *http.Request) (string, error) {
	err := m.runServiceCmd("stop")
	msg := ""
	if err == nil {
		msg = "Successfully sent stop request"
	}
	return msg, err
}

func (m *MonitClient) statusLookup(s MonitStatus) (string, error) {
	var tagForService ServiceTag
	foundService := false
	for _, serviceTag := range s.Services {
		if serviceTag.Name == m.monitConfig.ServiceName {
			tagForService = serviceTag
			foundService = true
			break
		}
	}
	if foundService == false {
		return "", fmt.Errorf("Could not find process %s", m.monitConfig.ServiceName)
	}

	switch {
	case tagForService.PendingAction != 0:
		return "pending", nil
	case tagForService.Monitor == 0:
		return "stopped", nil
	case tagForService.Monitor == 2:
		return "starting", nil
	case tagForService.Status == 0:
		return "running", nil
	default:
		return "failing", nil
	}
}

func (m *MonitClient) GetStatus(req *http.Request) (string, error) {
	statusResponse, err := m.runStatusCmd()
	if err != nil {
		return "", err
	}

	monitStatus, err := ParseXML(statusResponse)
	if err != nil {
		return "", err
	}

	status, err := m.statusLookup(monitStatus)
	if err != nil {
		return "", err
	}

	return status, nil
}

func (m *MonitClient) newUrl(endpoint string, queryParams ...url.Values) (*url.URL, error) {
	config := m.monitConfig

	statusURL, err := url.Parse(fmt.Sprintf("http://%s:%d/%s", config.Host, config.Port, endpoint))
	if err != nil {
		m.logger.Error("Failed to parse URL", err)
		m.logger.Info("URL info", lager.Data{
			"URL": statusURL,
		})
		return nil, err
	}

	if len(queryParams) > 0 {
		statusURL.RawQuery = queryParams[0].Encode()
	}

	return statusURL, nil
}

func (m *MonitClient) runStatusCmd() (io.Reader, error) {
	statusURL, err := m.newUrl("_status", url.Values{
		"format": []string{"xml"},
	})

	resp, err := m.sendRequest(statusURL, "GET")
	if err != nil {
		return nil, err
	}

	return resp, err
}

func (m *MonitClient) runServiceCmd(command string) error {
	serviceAction := fmt.Sprintf("action=%s", command)
	pendingStatusMsg := fmt.Sprintf("%s pending", command)
	statusURL, err := m.newUrl(m.monitConfig.ServiceName)

	respBody, err := m.sendRequest(statusURL, "POST", serviceAction)

	if err != nil {
		return err
	}
	responseBytes, _ := ioutil.ReadAll(respBody)
	responseStr := string(responseBytes)

	if !strings.Contains(responseStr, pendingStatusMsg) {
		monitFailure := fmt.Errorf("Monit failed to %s %s successfully", command, m.monitConfig.ServiceName)
		m.logger.Error("Monit failure:", monitFailure)
		m.logger.Info("request info", lager.Data{
			"response_body": string(responseBytes),
		})

		return monitFailure
	}

	return nil
}

func (m *MonitClient) sendRequest(statusURL *url.URL, reqMethod string, params ...string) (io.Reader, error) {
	config := m.monitConfig
	client := &http.Client{}

	var err error
	var req *http.Request
	if len(params) > 0 {
		req, err = http.NewRequest(reqMethod, statusURL.String(), strings.NewReader(params[0])) //bytes.NewBufferString(params[0]))
	} else {
		req, err = http.NewRequest(reqMethod, statusURL.String(), nil)
	}

	if err != nil {
		m.logger.Error("Failed to create http request", err)
		return nil, err
	}

	if reqMethod == "POST" || reqMethod == "PUT" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	m.logger.Info("Forwarding request to monit API", lager.Data{
		"url": req.URL,
	})

	req.SetBasicAuth(config.User, config.Password)

	resp, err := client.Do(req)
	if err != nil {
		errMsg := fmt.Errorf("Error sending http request: %s", err.Error())
		m.logger.Error(errMsg.Error(), err)
		m.logger.Info("request info", lager.Data{
			"request": req.URL,
		})
		return nil, errMsg
	}

	if resp.StatusCode != 200 {
		responseBytes, _ := ioutil.ReadAll(resp.Body)
		non200Error := fmt.Errorf("Received %d response from monit: %s", resp.StatusCode, responseBytes)
		m.logger.Error("Failed with non-200 response", non200Error)
		m.logger.Info("", lager.Data{
			"status_code":   resp.StatusCode,
			"response_body": string(responseBytes),
		})
		return nil, non200Error
	}

	m.logger.Info("Made successful request to monit API")
	return resp.Body, nil
}
