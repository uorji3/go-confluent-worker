package confluent

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/uorji3/go-confluent-worker/app/config"
	"github.com/uorji3/go-confluent-worker/app/logger"
)

const (
	confluentBaseURL = "https://api.telemetry.confluent.cloud"
	helpPrefix       = "# HELP "
	typePrefix       = "# TYPE "
)

type (
	Client struct {
		key               string
		secret            string
		objectResourceIDs map[string][]string
		objectMetricNames map[string][]string
	}

	ErrorResponse struct {
		Errors []Error `json:"errors"`
	}

	Error struct {
		Status string
		Detail string
	}

	MetricsResponse struct {
		Metrics []*Metric
	}

	Metric struct {
		Name         string
		Description  string
		Type         string
		Measurements []*Measurement
	}

	Measurement struct {
		Labels    []*Label
		Value     int64
		Timestamp time.Time
	}

	Label struct {
		Key   string
		Value string
	}

	plainTextResponse struct {
		Text string
	}
)

func (m Measurement) LabelMap() map[string]string {
	labelMap := make(map[string]string)
	for _, label := range m.Labels {
		labelMap[label.Key] = label.Value
	}

	return labelMap
}

func NewConfluentClient(config config.Config) *Client {

	objectResourceIDs := make(map[string][]string)
	objectMetricNames := make(map[string][]string)

	for _, resource := range config.Resources {
		metricNames := make([]string, 0)
		uniqueResourceIDs := make(map[string]bool)
		resourceKey := fmt.Sprintf("%s_id", resource.ResourceName)

		for _, metric := range resource.Metrics {
			metricNames = append(metricNames, metric.MetricName)
			for _, filter := range metric.Filters {
				for _, label := range filter.Labels {
					if label.Key == resourceKey {
						uniqueResourceIDs[label.Value] = true
					}
				}
			}
		}

		resourceIDs := make([]string, 0)
		for resourceID := range uniqueResourceIDs {
			resourceIDs = append(resourceIDs, resourceID)
		}

		objectResourceIDs[resource.ResourceName] = resourceIDs
		objectMetricNames[resource.ResourceName] = metricNames
	}

	return &Client{
		key:               config.Environment.ConfluentMetricsApiKey,
		secret:            config.Environment.ConfluentMetricsApiSecret,
		objectResourceIDs: objectResourceIDs,
		objectMetricNames: objectMetricNames,
	}
}

func (c *Client) CloudDatasetExport() (*MetricsResponse, error) {
	params := make(url.Values)

	for resourceName := range config.ObjectModel {
		resourceIDs := c.objectResourceIDs[resourceName]
		if len(resourceIDs) == 0 {
			continue
		}

		for _, resourceID := range resourceIDs {
			params.Add(fmt.Sprintf("resource.%s.id", resourceName), resourceID)
		}
	}

	response := &MetricsResponse{
		Metrics: make([]*Metric, 0),
	}

	var textResponse plainTextResponse
	errorResponse, err := c.do(http.MethodGet, "/v2/metrics/cloud/export", params, nil, &textResponse)
	if err != nil {
		logger.Errorf("Failed to get cloud dataset export errorResponse: %+v, err: %v", errorResponse, err)
		return response, err
	}

	err = c.populateMetricsResponse(response, textResponse.Text)
	return response, err
}

func (c *Client) populateMetricsResponse(response *MetricsResponse, text string) error {
	scanner := bufio.NewScanner(strings.NewReader(text))

	metric := &Metric{}

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, helpPrefix) {
			if metric.Name != "" {
				response.Metrics = append(response.Metrics, metric)
				metric = &Metric{}
			}

			line = strings.TrimPrefix(line, helpPrefix)
			spaceIndex := strings.Index(line, " ")
			if spaceIndex < 0 {
				return errors.New("cannot properly parse help line")
			}

			metric.Name = line[0:spaceIndex]
			metric.Description = line[spaceIndex+1:]
			metric.Measurements = make([]*Measurement, 0)
		} else if strings.HasPrefix(line, typePrefix) {
			spaceIndex := strings.LastIndex(line, " ")
			if spaceIndex < 0 {
				return errors.New("cannot properly parse type line")
			}

			metric.Type = line[spaceIndex+1:]
		} else {
			measurement := &Measurement{
				Labels: make([]*Label, 0),
			}

			leftCurlyBraceIndex := strings.Index(line, "{")
			righCurlyBraceIndex := strings.Index(line, "}")
			if leftCurlyBraceIndex < 1 || righCurlyBraceIndex < 1 {
				return errors.New("cannot properly parse labels in measurement line")
			}

			labelLine := line[leftCurlyBraceIndex+1 : righCurlyBraceIndex]
			labelLine = strings.TrimSuffix(labelLine, ",")

			labelParts := strings.Split(labelLine, ",")
			for _, labelPart := range labelParts {
				index := strings.Index(labelPart, "=")
				if index < 1 {
					return errors.New("cannot properly parse label parts in measurement line")
				}

				label := &Label{
					Key:   labelPart[0:index],
					Value: strings.TrimSuffix(strings.TrimPrefix(labelPart[index+1:], "\""), "\""),
				}

				measurement.Labels = append(measurement.Labels, label)
			}

			line = strings.TrimSpace(line[righCurlyBraceIndex+1:])
			index := strings.LastIndex(line, " ")
			if index < 1 {
				return errors.New("cannot properly parse value and timestamp in measurement line")
			}

			valueFloat, err := strconv.ParseFloat(line[0:index], 64)
			if err != nil {
				return err
			}
			measurement.Value = int64(valueFloat)

			timestampMillis, err := strconv.ParseInt(line[index+1:], 10, 64)
			if err != nil {
				return err
			}

			measurement.Timestamp = time.Unix(timestampMillis/1000, (timestampMillis%1000)*1e9)

			metric.Measurements = append(metric.Measurements, measurement)
		}
	}

	if metric.Name != "" {
		response.Metrics = append(response.Metrics, metric)
	}

	return scanner.Err()
}

func (c *Client) do(method, relativeURL string, params url.Values, payload interface{}, container interface{}) (*ErrorResponse, error) {

	var errorResponse ErrorResponse

	if params != nil {
		relativeURL += "?" + params.Encode()
	}

	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return &errorResponse, err
		}

		body = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, confluentBaseURL+relativeURL, body)
	if err != nil {
		return &errorResponse, err
	}

	req.Close = true
	req.SetBasicAuth(c.key, c.secret)
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return &errorResponse, err
	}

	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		err = json.NewDecoder(res.Body).Decode(&errorResponse)
		if err != nil {
			return &errorResponse, err
		}

		return &errorResponse, fmt.Errorf("invalid status code")
	}

	if strings.Contains(res.Header.Get("Content-Type"), "text/plain") {
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return &errorResponse, err
		}

		response, ok := container.(*plainTextResponse)
		if !ok {
			return &errorResponse, fmt.Errorf("cannot cast var to plain text response")
		}

		response.Text = string(b)
		return &errorResponse, nil
	} else {
		err = json.NewDecoder(res.Body).Decode(container)
		return &errorResponse, err
	}
}
