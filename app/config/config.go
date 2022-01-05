package config

import (
	"errors"
	"fmt"

	"github.com/uorji3/go-confluent-worker/app/util"
)

const metricTypePrefix = "custom.googleapis.com"

type (
	Config struct {
		Environment Environment `yaml:"environment"`
		Resources   []Resource  `yaml:"resources"`
	}

	Environment struct {
		ConfluentMetricsApiKey       string `yaml:"CONFLUENT_METRICS_API_KEY" json:"-"`
		ConfluentMetricsApiSecret    string `yaml:"CONFLUENT_METRICS_API_SECRET" json:"-"`
		DisableStdOutLogger          bool   `yaml:"DISABLE_STDOUT_LOGGER"`
		EnableGCPLogger              bool   `yaml:"ENABLE_GCP_LOGGER"`
		Environment                  string `yaml:"ENVIRONMENT"`
		GCPLoggerName                string `yaml:"GCP_LOGGER_NAME"`
		GoogleApplicationCredentials string `yaml:"GOOGLE_APPLICATION_CREDENTIALS" json:"-"`
		MetricNamespace              string `yaml:"METRIC_NAMESPACE"`
		Port                         string `yaml:"PORT"`
		SentryDSN                    string `yaml:"SENTRY_DSN" json:"-"`
	}

	Resource struct {
		ResourceName string   `yaml:"resource_name"`
		Metrics      []Metric `yaml:"metrics"`
	}

	Metric struct {
		MetricName string   `yaml:"metric_name"`
		Unit       string   `yaml:"unit"`
		Filters    []Filter `yaml:"filters"`
	}

	Filter struct {
		Labels []Label `yaml:"labels"`
		Suffix string  `yaml:"suffix"`
	}

	Label struct {
		Key   string `yaml:"key"`
		Value string `yaml:"value"`
	}
)

func (c Config) MetricTypePrefix() string {
	return metricTypePrefix
}

func (c Config) ResolvedMetricNamespace() string {
	if c.Environment.MetricNamespace != "" {
		return c.Environment.MetricNamespace
	}

	return "confluent"
}

func (c Config) Validate() error {
	if c.Environment.ConfluentMetricsApiKey == "" {
		return errors.New("must provide Confluent metrics api key")
	}

	if c.Environment.ConfluentMetricsApiSecret == "" {
		return errors.New("must provide Confluent metrics api secret")
	}

	if c.Environment.GoogleApplicationCredentials == "" {
		return errors.New("must provide Google application credentials")
	}

	if len(c.Resources) == 0 {
		return errors.New("must provide some resources")
	}

	// invert object map
	invertedObjectModel := make(map[string]string)
	invertedLabelsMap := make(map[string]map[string]bool)

	for resourceName, metricModels := range ObjectModel {
		for _, metricModel := range metricModels {
			invertedObjectModel[metricModel.Name] = resourceName
			labelMap := make(map[string]bool)
			for _, label := range metricModel.Labels {
				labelMap[label] = true
			}

			invertedLabelsMap[metricModel.Name] = labelMap
		}
	}

	visitedResources := make(map[string]bool)

	for _, resource := range c.Resources {
		if resource.ResourceName == "" {
			return errors.New("missing resource name")
		}

		if _, ok := ObjectModel[resource.ResourceName]; !ok {
			return fmt.Errorf("invalid resource name: %v", resource.ResourceName)
		}

		if visitedResources[resource.ResourceName] {
			return fmt.Errorf("duplicate resource name: %v", resource.ResourceName)
		} else {
			visitedResources[resource.ResourceName] = true
		}

		if len(resource.Metrics) == 0 {
			return fmt.Errorf("must provide at least one metric for resource: %v", resource.ResourceName)
		}

		visitedMetricNames := make(map[string]bool)
		for _, metric := range resource.Metrics {
			if metric.MetricName == "" {
				return fmt.Errorf("missing metric name for resource: %v", resource.ResourceName)
			}

			if visitedMetricNames[metric.MetricName] {
				return fmt.Errorf("duplicate metric name: %v", metric.MetricName)
			} else {
				visitedMetricNames[metric.MetricName] = true
			}

			// See: https://cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.metricDescriptors#MetricDescriptor
			if metric.Unit != "" {
				if metric.Unit != "bit" &&
					metric.Unit != "byte" &&
					metric.Unit != "second" &&
					metric.Unit != "minute" &&
					metric.Unit != "hour" &&
					metric.Unit != "day" &&
					metric.Unit != "dimensionless" {
					return fmt.Errorf("invalid unit: %v", metric.Unit)
				}
			}

			resourceName, ok := invertedObjectModel[metric.MetricName]
			if !ok {
				return fmt.Errorf("invalid metric name %v", metric.MetricName)
			}

			if resource.ResourceName != resourceName {
				return fmt.Errorf("invalid metric name %v for resource: %v", metric.MetricName, resource.ResourceName)
			}

			objectModelLabelMap, ok := invertedLabelsMap[metric.MetricName]
			if !ok {
				return fmt.Errorf("missing object models labels for metric: %v", metric.MetricName)
			}

			visitedMetricTypes := make(map[string]bool)
			for _, filter := range metric.Filters {
				if filter.Suffix == "" {
					return fmt.Errorf("missing filter suffix for metric: %v", metric.MetricName)
				}

				visitedFilterLabelKeys := make(map[string]bool)
				for _, filterLabel := range filter.Labels {
					if visitedFilterLabelKeys[filterLabel.Key] {
						return fmt.Errorf("duplicate filter %v for metric: %v", filterLabel.Key, metric.MetricName)
					}

					visitedFilterLabelKeys[filterLabel.Key] = true
				}

				for objectModelLabel := range objectModelLabelMap {
					if !visitedFilterLabelKeys[objectModelLabel] {
						return fmt.Errorf("missing filter label %v for metric: %v", objectModelLabel, metric.MetricName)
					}
				}

				for configFilterLabel := range visitedFilterLabelKeys {
					if !objectModelLabelMap[configFilterLabel] {
						return fmt.Errorf("invalid filter label %v for metric: %v", configFilterLabel, metric.MetricName)
					}
				}

				metricType := util.GenerateMetricType(metricTypePrefix, c.ResolvedMetricNamespace(), metric.MetricName, filter.Suffix)
				if len(metricType) > 100 {
					return fmt.Errorf("length of metric type %v for metric %v greater than 100 characters", metricType, metric.MetricName)
				}

				if visitedMetricTypes[metricType] {
					return fmt.Errorf("duplicate metric type: %v", metricType)
				} else {
					visitedMetricTypes[metricType] = true
				}
			}
		}
	}

	return nil
}
