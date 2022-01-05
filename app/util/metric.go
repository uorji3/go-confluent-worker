package util

import (
	"fmt"
)

func GenerateMetricType(metricTypePrefix, metricNamespace, metricName, suffix string) string {
	return fmt.Sprintf("%s/%s/%s_%s", metricTypePrefix, metricNamespace, metricName, suffix)
}
