package scraper

import (
	"context"
	"time"

	"github.com/uorji3/go-confluent-worker/app/config"
	"github.com/uorji3/go-confluent-worker/app/confluent"
	"github.com/uorji3/go-confluent-worker/app/logger"
	"github.com/uorji3/go-confluent-worker/app/metrics"
)

type Scraper struct {
	configMetricTypeMap map[string]bool
	configMetricUnitMap map[string]string
	confluentClient     *confluent.Client
	customMetricMap     map[string]bool
	metricsClient       *metrics.Client
	skippedMetricTypes  map[string]bool
}

func NewScraper(ctx context.Context, configBundle config.Config) (*Scraper, error) {

	metricFilterMap := make(map[string][]config.Filter)
	for _, resource := range configBundle.Resources {
		for _, metric := range resource.Metrics {
			metricFilterMap[metric.MetricName] = append(metricFilterMap[metric.MetricName], metric.Filters...)
		}
	}

	metricsClient, err := metrics.NewClient(ctx, configBundle.Environment.GoogleApplicationCredentials, metricFilterMap, configBundle.MetricTypePrefix(), configBundle.ResolvedMetricNamespace())
	if err != nil {
		return nil, err
	}

	customMetricMap, err := metricsClient.CustomMetricMap(ctx)
	if err != nil {
		return nil, err
	}

	configMetricTypeMap := make(map[string]bool)
	configMetricUnitMap := make(map[string]string)

	for _, resource := range configBundle.Resources {
		for _, metric := range resource.Metrics {
			if metric.Unit != "" {
				configMetricUnitMap[metric.MetricName] = metric.Unit
			}

			for _, filter := range metric.Filters {
				labelMap := make(map[string]string)
				for _, label := range filter.Labels {
					labelMap[label.Key] = label.Value
				}

				metricType, ok := metricsClient.GetMetricType(metric.MetricName, labelMap)
				if !ok {
					continue
				}

				configMetricTypeMap[metricType] = true
			}
		}
	}

	confluentClient := confluent.NewConfluentClient(configBundle)

	s := &Scraper{
		configMetricTypeMap: configMetricTypeMap,
		configMetricUnitMap: configMetricUnitMap,
		confluentClient:     confluentClient,
		customMetricMap:     customMetricMap,
		metricsClient:       metricsClient,
		skippedMetricTypes:  make(map[string]bool),
	}

	return s, nil
}

func (s *Scraper) Close() error {
	if s.metricsClient != nil {
		return s.metricsClient.Close()
	}

	return nil
}

func (s *Scraper) Run(ctx context.Context) error {
	logger.Infof("[Scraper] Scraping metrics...")

	// trigger a manual run soon after scraper runs
	go func() {
		time.Sleep(5 * time.Second)
		s.scrape(ctx)
	}()

	ticker := time.NewTicker(time.Duration(1) * time.Minute)

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case <-ticker.C:
			s.scrape(ctx)
		}
	}

	logger.Info("[Scraper] Gracefully terminated")

	return nil
}

func (s *Scraper) scrape(ctx context.Context) {
	t := time.Now()

	logger.Debugf("[Scraper] Scraping metrics at %v", t)

	metricsResponse, err := s.confluentClient.CloudDatasetExport()
	if err != nil {
		logger.Errorf("[Scraper] Failed to scrape metrics at time %v: %v", t, err)
		return
	}

	for _, metric := range metricsResponse.Metrics {
		metricUnit := s.configMetricUnitMap[metric.Name]
		for _, measurement := range metric.Measurements {

			metricType, ok := s.metricsClient.GetMetricType(metric.Name, measurement.LabelMap())
			if !ok {
				s.skippedMetricTypes[metricType] = true
				continue
			}

			if s.skippedMetricTypes[metricType] {
				continue
			}

			if !s.configMetricTypeMap[metricType] {
				s.skippedMetricTypes[metricType] = true
				continue
			}

			if !s.customMetricMap[metric.Name] {
				err := s.metricsClient.CreateCustomMetric(ctx, metric.Name, metric.Description, metricUnit, measurement)
				if err != nil {
					s.skippedMetricTypes[metric.Name] = true
					logger.Errorf("failed to create custom metric %v: %v", metric.Name, err)
					continue
				}
			}

			err = s.metricsClient.WriteCustomMetric(ctx, metric.Name, measurement)
			if err != nil {
				logger.Errorf("failed to write custom metric %v: %v", metricType, err)
			}
		}
	}

	logger.Debugf("[Scraper] Done scraping metrics at %v", t)
}
