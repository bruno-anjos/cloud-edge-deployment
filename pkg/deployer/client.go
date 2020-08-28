package deployer

import (
	"fmt"
	"net/http"
	"os"
	"time"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type DeployerClient struct {
	*utils.GenericClient
}

const (
	HeartbeatCheckerTimeout = 60
)

func NewDeployerClient(addr string) *DeployerClient {
	return &DeployerClient{
		GenericClient: utils.NewGenericClient(addr, Port),
	}
}

func (c *DeployerClient) ExpandTree(serviceId, location string) (status int) {
	var reqBody api.ExpandTreeRequestBody
	reqBody = location

	path := deployer.GetExpandTreePath(serviceId)
	req := utils.BuildRequest(http.MethodPost, c.HostPort, path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *DeployerClient) GetServices() (serviceIds []string, status int) {
	path := deployer.GetDeploymentsPath()
	req := utils.BuildRequest(http.MethodGet, c.HostPort, path, nil)

	var resp api.GetDeploymentsResponseBody
	status, _ = utils.DoRequest(c.Client, req, &resp)
	serviceIds = resp

	return
}

func (c *DeployerClient) RegisterService(serviceId string, static bool,
	deploymentYamlBytes []byte) (status int) {
	reqBody := api.RegisterServiceRequestBody{
		DeploymentId:        serviceId,
		Static:              static,
		DeploymentYAMLBytes: deploymentYamlBytes,
	}
	path := deployer.GetServicePath(serviceId)
	req := utils.BuildRequest(http.MethodPost, c.HostPort, path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *DeployerClient) DeleteService(serviceId string) (status int) {
	path := deployer.GetServicePath(serviceId)
	req := utils.BuildRequest(http.MethodDelete, c.HostPort, path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *DeployerClient) RegisterServiceInstance(serviceId, instanceId string, static bool,
	portTranslation nat.PortMap, local bool) (status int) {
	reqBody := api.RegisterServiceInstanceRequestBody{
		Static:          static,
		PortTranslation: portTranslation,
		Local:           local,
	}
	path := deployer.GetServiceInstancePath(serviceId, instanceId)
	req := utils.BuildRequest(http.MethodPost, c.HostPort, path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *DeployerClient) RegisterHearbeatServiceInstance(serviceId, instanceId string) (status int) {
	path := deployer.GetServiceInstanceAlivePath(serviceId, instanceId)
	req := utils.BuildRequest(http.MethodPost, c.HostPort, path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *DeployerClient) SendHearbeatServiceInstance(serviceId, instanceId string) (status int) {
	path := deployer.GetServiceInstanceAlivePath(serviceId, instanceId)
	req := utils.BuildRequest(http.MethodPut, c.HostPort, path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *DeployerClient) WarnOfDeadChild(serviceId, deadChildId string, grandChild *utils.Node) (status int) {
	var reqBody api.DeadChildRequestBody
	reqBody = *grandChild

	path := deployer.GetDeadChildPath(serviceId, deadChildId)
	req := utils.BuildRequest(http.MethodPost, c.HostPort, path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *DeployerClient) WarnToTakeChild(serviceId string, child *utils.Node) (status int) {
	var reqBody api.TakeChildRequestBody
	reqBody = *child

	path := deployer.GetTakeChildPath(serviceId)
	req := utils.BuildRequest(http.MethodPost, c.HostPort, path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *DeployerClient) WarnThatIAmParent(serviceId string, parent *utils.Node) (status int) {
	var reqBody api.IAmYourParentRequestBody
	reqBody = *parent

	path := deployer.GetImYourParentPath(serviceId)
	req := utils.BuildRequest(http.MethodPost, c.HostPort, path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *DeployerClient) GetHierarchyTable() (table map[string]*HierarchyEntryDTO, status int) {
	path := deployer.GetHierarchyTablePath()
	req := utils.BuildRequest(http.MethodGet, c.HostPort, path, nil)

	var resp api.GetHierarchyTableResponseBody
	status, _ = utils.DoRequest(c.Client, req, &resp)

	table = resp

	return
}

func (c *DeployerClient) SetParentAlive(parentId string) (status int) {
	path := deployer.GetParentAlivePath(parentId)
	req := utils.BuildRequest(http.MethodPost, c.HostPort, path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *DeployerClient) AddNode(nodeAddr string) (status int) {
	var reqBody api.AddNodeRequestBody
	reqBody = nodeAddr

	path := deployer.GetAddNodePath()
	req := utils.BuildRequest(http.MethodPost, c.HostPort, path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *DeployerClient) SendInstanceHeartbeatToDeployerPeriodically() {
	serviceId := os.Getenv(utils.ServiceEnvVarName)
	instanceId := os.Getenv(utils.InstanceEnvVarName)

	status := c.RegisterHearbeatServiceInstance(serviceId, instanceId)
	switch status {
	case http.StatusConflict:
		log.Debugf("service %s instance %s already has a heartbeat sender", serviceId, instanceId)
		return
	case http.StatusOK:
	default:
		panic(errors.New(fmt.Sprintf("received unexpected status %d", status)))
	}

	ticker := time.NewTicker((HeartbeatCheckerTimeout / 3) * time.Second)
	for {
		<-ticker.C
		log.Info("sending heartbeat to deployer")
		status = c.SendHearbeatServiceInstance(serviceId, instanceId)
		switch status {
		case http.StatusNotFound:
			log.Warnf("heartbeat to deployer retrieved not found")
		case http.StatusOK:
		default:
			panic(errors.New(fmt.Sprintf("received unexpected status %d", status)))
		}
	}
}
