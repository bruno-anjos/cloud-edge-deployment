package metrics

import (
	"fmt"
)

const (
	MetricNodeAddr           = "METRIC_NODE_ADDR"
	MetricLocation           = "METRIC_LOCATION"
	MetricLocationInVicinity = "METRIC_LOCATION_VICINITY"

	// SERVICE METRICS
	metricNumberOfInstancesPerServiceId   = "METRIC_NUMBER_OF_INSTANCES_PER_SERVICE_%s"
	metricLoadPerServiceInChild           = "METRIC_LOAD_PER_SERVICE_%s_IN_CHILD_%s"
	metricLoadPerServiceInChildren        = "METRIC_LOAD_PER_SERVICE_%s_IN_CHILDREN"
	metricAggLoadPerServiceInChildren     = "METRIC_AGG_LOAD_PER_SERVICE_%s_IN_CHILDREN"
	metricClientLatencyPerService         = "METRIC_CLIENT_LATENCY_PER_SERVICE_%s"
	metricProcessingTimePerService        = "METRIC_PROCESSING_TIME_PER_SERVICE_%s"
	metricAverageClientLocationPerService = "METRIC_AVERAGE_CLIENT_LOCATION_PER_SERVICE_%s"
)

func GetNumInstancesMetricId(serviceId string) string {
	return fmt.Sprintf(metricNumberOfInstancesPerServiceId, serviceId)
}

func GetLoadPerServiceInChildMetricId(serviceId, childId string) string {
	return fmt.Sprintf(metricLoadPerServiceInChild, serviceId, childId)
}

func GetLoadPerServiceInChildrenMetricId(serviceId string) string {
	return fmt.Sprintf(metricLoadPerServiceInChildren, serviceId)
}

func GetAggLoadPerServiceInChildrenMetricId(serviceId string) string {
	return fmt.Sprintf(metricAggLoadPerServiceInChildren, serviceId)
}

func GetClientLatencyPerServiceMetricId(serviceId string) string {
	return fmt.Sprintf(metricClientLatencyPerService, serviceId)
}

func GetProcessingTimePerServiceMetricId(serviceId string) string {
	return fmt.Sprintf(metricProcessingTimePerService, serviceId)
}

func GetAverageClientLocationPerServiceMetricId(serviceId string) string {
	return fmt.Sprintf(metricAverageClientLocationPerService, serviceId)
}
