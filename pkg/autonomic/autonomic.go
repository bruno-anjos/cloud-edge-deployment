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
	GetID(addr string) (id string, status int)
	RegisterDeployment(addr, deploymentID, strategyID string, depthFactor float64, exploringTTL int) (status int)
	DeleteDeployment(addr, deploymentID string) (status int)
	GetDeployments(addr string) (deployments map[string]*autonomic.DeploymentDTO, status int)
	AddDeploymentChild(addr, deploymentID string, child *utils.Node) (status int)
	RemoveDeploymentChild(addr, deploymentID, childID string) (status int)
	SetDeploymentParent(addr, deploymentID string, parent *utils.Node) (status int)
	IsNodeInVicinity(addr, nodeID string) (isInVicinity bool)
	GetClosestNode(addr string, locations []s2.CellID, toExclude map[string]interface{}) (closest *utils.Node)
	SetExploredSuccessfully(addr, deploymentID, childID string) (status int)
	BlacklistNodes(addr, deploymentID, origin string, nodes []string, nodesVisited map[string]struct{}) (status int)
}

type ClientFactory interface {
	New() Client
}
