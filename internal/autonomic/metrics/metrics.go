package metrics

import (
	"fmt"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

type (
	VicinityMetric struct {
		Nodes     map[string]*utils.Node
		Locations map[string]string
	}
)

const (
	MetricNodeAddr           = "METRIC_NODE_ADDR"
	MetricLocation           = "METRIC_LOCATION"
	MetricLocationInVicinity = "METRIC_LOCATION_VICINITY"

	// DEPLOYMENT METRICS
	metricNumberOfInstancesPerDeploymentId   = "METRIC_NUMBER_OF_INSTANCES_PER_DEPLOYMENT_%s"
	metricLoadPerDeploymentInChildren        = "METRIC_LOAD_PER_DEPLOYMENT_%s_IN_CHILDREN"
	metricAggLoadPerDeploymentInChildren     = "METRIC_AGG_LOAD_PER_DEPLOYMENT_%s_IN_CHILDREN"
	metricClientLatencyPerDeployment         = "METRIC_CLIENT_LATENCY_PER_DEPLOYMENT_%s"
	metricProcessingTimePerDeployment        = "METRIC_PROCESSING_TIME_PER_DEPLOYMENT_%s"
	metricAverageClientLocationPerDeployment = "METRIC_AVERAGE_CLIENT_LOCATION_PER_DEPLOYMENT_%s"
	metricLoadPerDeployment                  = "METRIC_LOAD_PER_DEPLOYMENT_%s"
)

func GetNumInstancesMetricId(deploymentId string) string {
	return fmt.Sprintf(metricNumberOfInstancesPerDeploymentId, deploymentId)
}

func GetLoadPerDeploymentInChildrenMetricId(deploymentId string) string {
	return fmt.Sprintf(metricLoadPerDeploymentInChildren, deploymentId)
}

func GetAggLoadPerDeploymentInChildrenMetricId(deploymentId string) string {
	return fmt.Sprintf(metricAggLoadPerDeploymentInChildren, deploymentId)
}

func GetClientLatencyPerDeploymentMetricId(deploymentId string) string {
	return fmt.Sprintf(metricClientLatencyPerDeployment, deploymentId)
}

func GetProcessingTimePerDeploymentMetricId(deploymentId string) string {
	return fmt.Sprintf(metricProcessingTimePerDeployment, deploymentId)
}

func GetAverageClientLocationPerDeploymentMetricId(deploymentId string) string {
	return fmt.Sprintf(metricAverageClientLocationPerDeployment, deploymentId)
}

func GetLoadPerDeployment(deploymentId string) string {
	return fmt.Sprintf(metricLoadPerDeployment, deploymentId)
}
