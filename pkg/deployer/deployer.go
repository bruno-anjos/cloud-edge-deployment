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
	GetDeployments() (deploymentIds []string, status int)
	RegisterDeployment(deploymentID string, static bool, deploymentYamlBytes []byte,
		grandparent, parent *utils.Node, children []*utils.Node, exploringTTL int) (status int)
	ExtendDeploymentTo(deploymentID string, node *utils.Node, exploringTTL int,
		config *deployer.ExtendDeploymentConfig) (status int)
	DeleteDeployment(deploymentID string) (status int)
	RegisterDeploymentInstance(deploymentID, instanceID string, static bool,
		portTranslation nat.PortMap, local bool) (status int)
	RegisterHearbeatDeploymentInstance(deploymentID, instanceID string) (status int)
	SendHearbeatDeploymentInstance(deploymentID, instanceID string) (status int)
	WarnOfDeadChild(deploymentID, deadChildID string, grandChild *utils.Node,
		alternatives map[string]*utils.Node, locations []s2.CellID) (status int)
	SetGrandparent(deploymentID string, grandparent *utils.Node) (status int)
	WarnThatIAmParent(deploymentID string, parent, grandparent *utils.Node) (status int)
	ChildDeletedDeployment(deploymentID, childID string) (status int)
	GetHierarchyTable() (table map[string]*deployer.HierarchyEntryDTO, status int)
	SetParentAlive(parentID string) (status int)
	SendInstanceHeartbeatToDeployerPeriodically()
	SendAlternatives(myID string, alternatives []*utils.Node) (status int)
	Fallback(deploymentID string, orphan *utils.Node, orphanLocation s2.CellID) (status int)
	GetFallback() (fallback *utils.Node, status int)
	HasDeployment(deploymentID string) (has bool, status int)
	PropagateLocationToHorizon(deploymentID string, origin *utils.Node, location s2.CellID, TTL int8,
		op deployer.PropagateOpType) (status int)
}

type ClientFactory interface {
	New(addr string) Client
}
