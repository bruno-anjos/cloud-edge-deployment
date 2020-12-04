package archimedes

import (
	api "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	internalUtils "github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/docker/go-connections/nat"
	"github.com/golang/geo/s2"
)

const (
	ArchimedesPort = internalUtils.ArchimedesPort
)

type Client interface {
	utils.GenericClient
	RegisterDeployment(deploymentId string, ports nat.PortSet, host *internalUtils.Node) (status int)
	RegisterDeploymentInstance(deploymentId, instanceId string, static bool,
		portTranslation nat.PortMap, local bool) (status int)
	DeleteDeployment(deploymentId string) (status int)
	DeleteDeploymentInstance(deploymentId, instanceId string) (status int)
	GetDeployments() (deployments map[string]*api.Deployment, status int)
	GetDeployment(deploymentId string) (instances map[string]*api.Instance, status int)
	GetDeploymentInstance(deploymentId, instanceId string) (instance *api.Instance, status int)
	GetInstance(instanceId string) (instance *api.Instance, status int)
	Resolve(host string, port nat.Port, deploymentId string, cLocation s2.CellID,
		reqId string) (rHost, rPort string, status int, timedOut bool)
	Redirect(deploymentId, target string, amount int) (status int)
	RemoveRedirect(deploymentId string) (status int)
	GetRedirected(deploymentId string) (redirected int32, status int)
	SetResolvingAnswer(id string, resolved *api.ResolvedDTO) (status int)
	GetLoad(deploymentId string) (load int, status int)
	GetClientCentroids(deploymentId string) (centroids []s2.CellID, status int)
	SetExploringCells(deploymentId string, cells []s2.CellID) (status int)
	AddDeploymentNode(deploymentId string, nodeId string, location s2.CellID,
		exploring bool) (status int)
	DeleteDeploymentNode(deploymentId string, nodeId string) (status int)
	CanRedirectToYou(deploymentId, nodeId string) (can bool, status int)
	WillRedirectToYou(deploymentId, nodeId string) (status int)
	StopRedirectingToYou(deploymentId, nodeId string) (status int)
}

type ClientFactory interface {
	New(addr string) Client
}
