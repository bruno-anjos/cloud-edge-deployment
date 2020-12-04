package deployer

import (
	"github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	internalUtils "github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/docker/go-connections/nat"
	"github.com/golang/geo/s2"
)

const (
	Port = internalUtils.DeployerPort
)

type Client interface {
	utils.GenericClient
	GetDeployments() (deploymentIds []string, status int)
	RegisterDeployment(deploymentId string, static bool, deploymentYamlBytes []byte,
		grandparent, parent *internalUtils.Node, children []*internalUtils.Node, exploringTTL int) (status int)
	ExtendDeploymentTo(deploymentId string, node *internalUtils.Node, exploringTTL int,
		config *deployer.ExtendDeploymentConfig) (status int)
	DeleteDeployment(deploymentId string) (status int)
	RegisterDeploymentInstance(deploymentId, instanceId string, static bool,
		portTranslation nat.PortMap, local bool) (status int)
	RegisterHearbeatDeploymentInstance(deploymentId, instanceId string) (status int)
	SendHearbeatDeploymentInstance(deploymentId, instanceId string) (status int)
	WarnOfDeadChild(deploymentId, deadChildId string, grandChild *internalUtils.Node,
		alternatives map[string]*internalUtils.Node, locations []s2.CellID) (status int)
	SetGrandparent(deploymentId string, grandparent *internalUtils.Node) (status int)
	WarnThatIAmParent(deploymentId string, parent, grandparent *internalUtils.Node) (status int)
	ChildDeletedDeployment(deploymentId, childId string) (status int)
	GetHierarchyTable() (table map[string]*deployer.HierarchyEntryDTO, status int)
	SetParentAlive(parentId string) (status int)
	SendInstanceHeartbeatToDeployerPeriodically()
	SendAlternatives(myId string, alternatives []*internalUtils.Node) (status int)
	Fallback(deploymentId string, orphan *internalUtils.Node, orphanLocation s2.CellID) (status int)
	GetFallback() (fallback *internalUtils.Node, status int)
	HasDeployment(deploymentId string) (has bool, status int)
	PropagateLocationToHorizon(deploymentId, origin string, location s2.CellID, TTL int8,
		op deployer.PropagateOpType) (status int)
}

type ClientFactory interface {
	New(addr string) Client
}
