package autonomic

import (
	"github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/golang/geo/s2"
)

const (
	StrategyIdealLatencyID = "STRATEGY_IDEAL_LATENCY"
	StrategyLoadBalanceID  = "STRATEGY_LOAD_BALANCE"

	Port = 50003
)

type Client interface {
	utils.GenericClient
	RegisterDeployment(deploymentID, strategyID string, depthFactor float64, exploringTTL int) (status int)
	DeleteDeployment(deploymentID string) (status int)
	GetDeployments() (deployments map[string]*autonomic.DeploymentDTO, status int)
	AddDeploymentChild(deploymentID string, child *utils.Node) (status int)
	RemoveDeploymentChild(deploymentID, childID string) (status int)
	SetDeploymentParent(deploymentID string, parent *utils.Node) (status int)
	IsNodeInVicinity(nodeID string) (isInVicinity bool)
	GetClosestNode(locations []s2.CellID, toExclude map[string]interface{}) (closest *utils.Node)
	GetVicinity() (vicinity *autonomic.Vicinity, status int)
	GetLocation() (location s2.CellID, status int)
	SetExploredSuccessfully(deploymentID, childID string) (status int)
	BlacklistNodes(deploymentID, origin string, nodes []string, nodesVisited map[string]struct{}) (status int)
}

type ClientFactory interface {
	New(addr string) Client
}
