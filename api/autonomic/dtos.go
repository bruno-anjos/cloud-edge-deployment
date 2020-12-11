package autonomic

import (
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
)

type deploymentConfig struct {
	DepthFactor  float64
	StrategyID   string
	ExploringTTL int
}

type DeploymentDTO struct {
	DeploymentID string
	StrategyID   string
	Children     []string
	Parent       *utils.Node
}
