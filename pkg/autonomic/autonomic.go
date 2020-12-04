package autonomic

import (
	"github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/golang/geo/s2"
)

const (
	StrategyIdealLatencyId = "STRATEGY_IDEAL_LATENCY"
	StrategyLoadBalanceId  = "STRATEGY_LOAD_BALANCE"

	Port = 50003
)

type Client interface {
	utils.GenericClient
	RegisterDeployment(deploymentId, strategyId string, depthFactor float64,
		exploringTTL int) (status int)
	DeleteDeployment(deploymentId string) (status int)
	GetDeployments() (deployments map[string]*autonomic.DeploymentDTO, status int)
	AddDeploymentChild(deploymentId string, child *utils.Node) (status int)
	RemoveDeploymentChild(deploymentId, childId string) (status int)
	SetDeploymentParent(deploymentId string, parent *utils.Node) (status int)
	IsNodeInVicinity(nodeId string) (isInVicinity bool)
	GetClosestNode(locations []s2.CellID, toExclude map[string]interface{}) (closest *utils.Node)
	GetVicinity() (vicinity *autonomic.Vicinity, status int)
	GetLocation() (location s2.CellID, status int)
	SetExploredSuccessfully(deploymentId, childId string) (status int)
	BlacklistNodes(deploymentId, origin string, nodes ...string) (status int)
}
type ClientFactory interface {
	New(addr string) Client
}
