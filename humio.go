package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strconv"

	"go.uber.org/zap"
)

// ErrQueryNotFound is returned when a query job no longer exists on the server
var ErrQueryNotFound = errors.New("query job not found")

type client struct {
	httpClient *http.Client
	token      string
	baseURL    string
}

type startQueryPayload struct {
	QueryString string `json:"queryString"`
	Start       string `json:"start"`
	End         string `json:"end"`
	IsLive      bool   `json:"isLive"`
}

func (c *client) startQueryJob(query, repo, metricName, start, end string, labels []MetricLabel) (queryJob, error) {
	postData := startQueryPayload{
		QueryString: query,
		Start:       start,
		End:         end,
		IsLive:      true,
	}

	var reader bytes.Buffer
	err := json.NewEncoder(&reader).Encode(&postData)
	if err != nil {
		return queryJob{}, err
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/repositories/%s/queryjobs", c.baseURL, repo), &reader)
	if err != nil {
		return queryJob{}, err
	}

	response, err := c.do(req)
	if err != nil {
		return queryJob{}, err
	}

	var queryResponse queryJob
	err = json.NewDecoder(response.Body).Decode(&queryResponse)
	if err != nil {
		return queryJob{}, err
	}
	queryResponse.Timespan = start
	queryResponse.Repo = repo
	queryResponse.MetricName = metricName
	queryResponse.MetricLabels = labels

	return queryResponse, nil
}

func (c *client) stopQueryJob(id, repo string) error {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/v1/repositories/%s/queryjobs/%s", c.baseURL, repo, id), nil)
	if err != nil {
		return err
	}
	_, err = c.do(req)
	if err != nil {
		return err
	}
	return nil
}

func (c *client) pollQueryJob(id, repo string) (queryJobData, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/repositories/%s/queryjobs/%s", c.baseURL, repo, id), nil)
	if err != nil {
		return queryJobData{}, err
	}

	response, err := c.do(req)
	if err != nil {
		return queryJobData{}, err
	}

	var queryJobDataResponse queryJobData
	err = json.NewDecoder(response.Body).Decode(&queryJobDataResponse)
	if err != nil {
		return queryJobData{}, err
	}

	return queryJobDataResponse, nil
}

func (c *client) do(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.token))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			zap.L().Sugar().Errorf("read body failed: %v", err)
			body = []byte("failed to read body")
		}

		// Check for 404 Not Found - query job may have expired
		if response.StatusCode == http.StatusNotFound {
			return nil, ErrQueryNotFound
		}

		requestDump, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			zap.L().Sugar().Debugf("Failed to dump request for logging")
		} else {
			zap.L().Sugar().Debugf("Failed request dump: %s", requestDump)
		}
		return nil, fmt.Errorf("request not OK: %s: body: %s", response.Status, body)
	}
	return response, nil
}

func parseFloat(input interface{}) (float64, error) {
	value, err := strconv.ParseFloat(input.(string), 64)
	if err != nil {
		return 0, err
	}
	return value, nil
}
