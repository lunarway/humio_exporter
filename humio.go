package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strconv"

	"github.com/prometheus/common/log"
)

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

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/dataspaces/%s/queryjobs/", c.baseURL, repo), &reader)
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
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/v1/dataspaces/%s/queryjobs/%s", c.baseURL, repo, id), nil)
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
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/dataspaces/%s/queryjobs/%s", c.baseURL, repo, id), nil)
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
			log.Errorf("read body failed: %v", err)
			body = []byte("failed to read body")
		}
		requestDump, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			log.Debugf("Failed to dump request for logging")
		} else {
			log.Debugf("Failed request dump: %s", requestDump)
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
