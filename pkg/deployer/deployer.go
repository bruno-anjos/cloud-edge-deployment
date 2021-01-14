package deployer

import (
	"github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/docker/go-connections/nat"
	"github.com/golang/geo/s2"
)

const (
	Port = 50002
)

type Client interface {
	utils.GenericClient
	GetDeployments(addr string) (deploymentIds []string, status int)
	RegisterDeployment(addr, deploymentID string, static bool, deploymentYamlBytes []byte,
		grandparent, parent *utils.Node, children []*utils.Node, exploringTTL int) (status int)
	ExtendDeploymentTo(addr, deploymentID string, node *utils.Node, exploringTTL int,
		config *deployer.ExtendDeploymentConfig) (status int)
	DeleteDeployment(addr, deploymentID string) (status int)
	RegisterDeploymentInstance(addr, deploymentID, instanceID string, static bool,
		portTranslation nat.PortMap, local bool) (status int)
	RegisterHearbeatDeploymentInstance(addr, deploymentID, instanceID string) (status int)
	SendHearbeatDeploymentInstance(addr, deploymentID, instanceID string) (status int)
	WarnOfDeadChild(addr, deploymentID, deadChildID string, grandChild *utils.Node,
		alternatives map[string]*utils.Node, locations []s2.CellID) (status int)
	SetGrandparent(addr, deploymentID string, grandparent *utils.Node) (status int)
	WarnThatIAmParent(addr, deploymentID string, parent, grandparent *utils.Node) (status int)
	ChildDeletedDeployment(addr, deploymentID, childID string) (status int)
	GetHierarchyTable(addr string) (table map[string]*deployer.HierarchyEntryDTO, status int)
	SetParentAlive(addr, parentID string) (status int)
	SendInstanceHeartbeatToDeployerPeriodically(addr string)
	SendAlternatives(addr, myID string, alternatives []*utils.Node) (status int)
	Fallback(addr, deploymentID string, orphan *utils.Node, orphanLocation s2.CellID) (status int)
	GetFallback(addr string) (fallback *utils.Node, status int)
	HasDeployment(addr, deploymentID string) (has bool, status int)
	PropagateLocationToHorizon(addr, deploymentID string, origin *utils.Node, location s2.CellID, TTL int8,
		op deployer.PropagateOpType) (status int)
	SetReady(addr string) (status int)
}

type ClientFactory interface {
	New() Client
}
