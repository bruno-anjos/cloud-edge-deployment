package metrics

import (
	"fmt"

	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
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
	metricNumberOfInstancesPerDeploymentID   = "METRIC_NUMBER_OF_INSTANCES_PER_DEPLOYMENT_%s"
	metricLoadPerDeploymentInChildren        = "METRIC_LOAD_PER_DEPLOYMENT_%s_IN_CHILDREN"
	metricAggLoadPerDeploymentInChildren     = "METRIC_AGG_LOAD_PER_DEPLOYMENT_%s_IN_CHILDREN"
	metricClientLatencyPerDeployment         = "METRIC_CLIENT_LATENCY_PER_DEPLOYMENT_%s"
	metricProcessingTimePerDeployment        = "METRIC_PROCESSING_TIME_PER_DEPLOYMENT_%s"
	metricAverageClientLocationPerDeployment = "METRIC_AVERAGE_CLIENT_LOCATION_PER_DEPLOYMENT_%s"
	metricLoadPerDeployment                  = "METRIC_LOAD_PER_DEPLOYMENT_%s"
)

func GetNumInstancesMetricID(deploymentID string) string {
	return fmt.Sprintf(metricNumberOfInstancesPerDeploymentID, deploymentID)
}

func GetLoadPerDeploymentInChildrenMetricID(deploymentID string) string {
	return fmt.Sprintf(metricLoadPerDeploymentInChildren, deploymentID)
}

func GetAggLoadPerDeploymentInChildrenMetricID(deploymentID string) string {
	return fmt.Sprintf(metricAggLoadPerDeploymentInChildren, deploymentID)
}

func GetClientLatencyPerDeploymentMetricID(deploymentID string) string {
	return fmt.Sprintf(metricClientLatencyPerDeployment, deploymentID)
}

func GetProcessingTimePerDeploymentMetricID(deploymentID string) string {
	return fmt.Sprintf(metricProcessingTimePerDeployment, deploymentID)
}

func GetAverageClientLocationPerDeploymentMetricID(deploymentID string) string {
	return fmt.Sprintf(metricAverageClientLocationPerDeployment, deploymentID)
}

func GetLoadPerDeployment(deploymentID string) string {
	return fmt.Sprintf(metricLoadPerDeployment, deploymentID)
}
