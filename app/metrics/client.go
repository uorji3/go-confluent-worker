package metrics

import (
	"context"
	"encoding/json"
	"fmt"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/uorji3/go-confluent-worker/app/config"
	"github.com/uorji3/go-confluent-worker/app/confluent"
	"github.com/uorji3/go-confluent-worker/app/util"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/api/label"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

type (
	Client struct {
		metricClient     *monitoring.MetricClient
		metricFilterMap  map[string][]config.Filter
		metricTypePrefix string
		metricNamespace  string
		projectID        string
	}
)

func NewClient(ctx context.Context, credentialsString string, metricFilterMap map[string][]config.Filter, metricTypePrefix, metricNamespace string) (*Client, error) {

	b := []byte(credentialsString)

	var credMap map[string]interface{}
	err := json.Unmarshal(b, &credMap)
	if err != nil {
		return nil, err
	}

	projectID := credMap["project_id"].(string)

	opts := option.WithCredentialsJSON(b)

	metricClient, err := monitoring.NewMetricClient(ctx, opts)
	if err != nil {
		return nil, err
	}

	return &Client{
		metricClient:     metricClient,
		metricFilterMap:  metricFilterMap,
		metricTypePrefix: metricTypePrefix,
		metricNamespace:  metricNamespace,
		projectID:        projectID,
	}, nil
}

func (c *Client) Close() error {
	return c.metricClient.Close()
}

func (c *Client) CreateCustomMetric(ctx context.Context, metricName, metricDescription, unit string, measurement *confluent.Measurement) error {

	metricType, ok := c.GetMetricType(metricName, measurement.LabelMap())
	if !ok {
		return fmt.Errorf("could not find filter for metric: %v", metricName)
	}

	labels := make([]*label.LabelDescriptor, len(measurement.Labels))
	for index, measurementLabel := range measurement.Labels {
		labels[index] = &label.LabelDescriptor{
			Key:       measurementLabel.Key,
			ValueType: label.LabelDescriptor_STRING,
		}
	}

	md := &metricpb.MetricDescriptor{
		Name:        metricName,
		Type:        metricType,
		Labels:      labels,
		MetricKind:  metricpb.MetricDescriptor_GAUGE,
		ValueType:   metricpb.MetricDescriptor_INT64,
		Description: metricDescription,
		DisplayName: metricName,
	}

	resolvedUnit := c.resolveUnit(unit)
	if resolvedUnit != "" {
		md.Unit = resolvedUnit
	}

	req := &monitoringpb.CreateMetricDescriptorRequest{
		Name:             "projects/" + c.projectID,
		MetricDescriptor: md,
	}

	_, err := c.metricClient.CreateMetricDescriptor(ctx, req)
	if err != nil {
		return fmt.Errorf("could not create custom metric: %v", metricType)
	}

	return nil
}

func (c *Client) CustomMetricMap(ctx context.Context) (map[string]bool, error) {
	nameMap := make(map[string]bool)

	req := &monitoringpb.ListMetricDescriptorsRequest{
		Name:   "projects/" + c.projectID,
		Filter: fmt.Sprintf("metric.type = starts_with(\"%s/%s\")", c.metricTypePrefix, c.metricNamespace),
	}

	iter := c.metricClient.ListMetricDescriptors(ctx, req)

	for {
		resp, err := iter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nameMap, err
		}

		nameMap[resp.Name] = true
	}

	return nameMap, nil
}

func (c *Client) WriteCustomMetric(ctx context.Context, metricName string, measurement *confluent.Measurement) error {

	metricType, ok := c.GetMetricType(metricName, measurement.LabelMap())
	if !ok {
		return fmt.Errorf("could not find filter for metric: %v", metricName)
	}

	labels := make(map[string]string)
	for _, measurementLabel := range measurement.Labels {
		labels[measurementLabel.Key] = labels[measurementLabel.Value]
	}

	measurementTimestamp := &timestamp.Timestamp{
		Seconds: measurement.Timestamp.Unix(),
	}

	timeSeries := []*monitoringpb.TimeSeries{
		{
			Metric: &metricpb.Metric{
				Type:   metricType,
				Labels: labels,
			},
			Points: []*monitoringpb.Point{
				{
					Interval: &monitoringpb.TimeInterval{
						StartTime: measurementTimestamp,
						EndTime:   measurementTimestamp,
					},
					Value: &monitoringpb.TypedValue{
						Value: &monitoringpb.TypedValue_Int64Value{
							Int64Value: measurement.Value,
						},
					},
				},
			},
		},
	}

	req := &monitoringpb.CreateTimeSeriesRequest{
		Name:       "projects/" + c.projectID,
		TimeSeries: timeSeries,
	}

	err := c.metricClient.CreateTimeSeries(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to write custom metric %v: %v ", metricType, err)
	}

	return nil
}

func (c *Client) GetMetricType(metricName string, labelMap map[string]string) (string, bool) {
	metricFilter, ok := c.findFilterForMeasurment(metricName, labelMap)
	if !ok {
		return "", false
	}

	return c.metricType(metricName, metricFilter.Suffix), true
}

func (c *Client) findFilterForMeasurment(metricName string, labelMap map[string]string) (config.Filter, bool) {
	metricFilters, ok := c.metricFilterMap[metricName]
	if !ok {
		return config.Filter{}, false
	}

	for _, metricFilter := range metricFilters {
		found := true
		for _, label := range metricFilter.Labels {
			if labelMap[label.Key] != label.Value {
				found = false
				break
			}
		}

		if found {
			return metricFilter, true
		}
	}

	return config.Filter{}, false
}

func (c *Client) metricType(metricName, suffix string) string {
	return util.GenerateMetricType(c.metricTypePrefix, c.metricNamespace, metricName, suffix)
}

func (c *Client) resolveUnit(unit string) string {
	switch unit {
	case "bit":
		return "bit"
	case "byte":
		return "By"
	case "second":
		return "s"
	case "minute":
		return "min"
	case "hour":
		return "h"
	case "day":
		return "d"
	case "dimensionless":
		return "1"
	default:
		return ""
	}
}
