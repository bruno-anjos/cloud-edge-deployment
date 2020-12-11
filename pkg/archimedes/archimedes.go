package archimedes

import (
	api "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/docker/go-connections/nat"
	"github.com/golang/geo/s2"
)

const (
	Port = 50000
)

type Client interface {
	utils.GenericClient
	RegisterDeployment(deploymentID string, ports nat.PortSet, host *utils.Node) (status int)
	RegisterDeploymentInstance(deploymentID, instanceID string, static bool,
		portTranslation nat.PortMap, local bool) (status int)
	DeleteDeployment(deploymentID string) (status int)
	DeleteDeploymentInstance(deploymentID, instanceID string) (status int)
	GetDeployments() (deployments map[string]*api.Deployment, status int)
	GetDeployment(deploymentID string) (instances map[string]*api.Instance, status int)
	GetDeploymentInstance(deploymentID, instanceID string) (instance *api.Instance, status int)
	GetInstance(instanceID string) (instance *api.Instance, status int)
	Resolve(host string, port nat.Port, deploymentID string, cLocation s2.CellID,
		reqID string) (rHost, rPort string, status int, timedOut bool)
	Redirect(deploymentID, target string, amount int) (status int)
	RemoveRedirect(deploymentID string) (status int)
	GetRedirected(deploymentID string) (redirected int32, status int)
	SetResolvingAnswer(id string, resolved *api.ResolvedDTO) (status int)
	GetLoad(deploymentID string) (load int, status int)
	GetClientCentroids(deploymentID string) (centroids []s2.CellID, status int)
	SetExploringCells(deploymentID string, cells []s2.CellID) (status int)
	AddDeploymentNode(deploymentID string, node *utils.Node, location s2.CellID,
		exploring bool) (status int)
	DeleteDeploymentNode(deploymentID string, nodeID string) (status int)
	CanRedirectToYou(deploymentID, nodeID string) (can bool, status int)
	WillRedirectToYou(deploymentID, nodeID string) (status int)
	StopRedirectingToYou(deploymentID, nodeID string) (status int)
}

type ClientFactory interface {
	New(addr string) Client
}
