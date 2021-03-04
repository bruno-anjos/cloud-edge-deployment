package archimedes

import (
	api "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/docker/go-connections/nat"
	"github.com/golang/geo/s2"
)

const (
	Port = 1500
)

type Client interface {
	utils.GenericClient
	RegisterDeployment(addr, deploymentID string, ports nat.PortSet, host *utils.Node) (status int)
	RegisterDeploymentInstance(addr, deploymentID, instanceID string, static bool,
		portTranslation nat.PortMap, local bool) (status int)
	DeleteDeployment(addr, deploymentID string) (status int)
	DeleteDeploymentInstance(addr, deploymentID, instanceID string) (status int)
	GetDeployments(addr string) (deployments map[string]*api.Deployment, status int)
	GetDeployment(addr, deploymentID string) (instances map[string]*api.Instance, status int)
	GetDeploymentInstance(addr, deploymentID, instanceID string) (instance *api.Instance, status int)
	GetInstance(addr, instanceID string) (instance *api.Instance, status int)
	Resolve(host string, port nat.Port, deploymentID string, cLocation s2.CellID,
		reqID string) (rHost, rPort string, status int, timedOut bool)
	Redirect(addr, deploymentID, target string, amount int) (status int)
	RemoveRedirect(addr, deploymentID string) (status int)
	GetRedirected(addr, deploymentID string) (redirected int32, status int)
	SetResolvingAnswer(addr, id string, resolved *api.ResolvedDTO) (status int)
	SetExploringCells(addr, deploymentID string, cells []s2.CellID) (status int)
	AddDeploymentNode(addr, deploymentID string, node *utils.Node, location s2.CellID,
		exploring bool) (status int)
	DeleteDeploymentNode(addr, deploymentID, nodeID string) (status int)
	CanRedirectToYou(addr, deploymentID, nodeID string) (can bool, status int)
	WillRedirectToYou(addr, deploymentID, nodeID string) (status int)
	StopRedirectingToYou(addr, deploymentID, nodeID string) (status int)
}

type ClientFactory interface {
	New(addr string) Client
}
