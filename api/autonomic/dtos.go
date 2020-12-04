package autonomic

import (
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
)

type deploymentConfig struct {
	DepthFactor  float64
	StrategyId   string
	ExploringTTL int
}

type DeploymentDTO struct {
	DeploymentId string
	StrategyId   string
	Children     []string
	Parent       *utils.Node
}
