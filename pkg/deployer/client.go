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

type Client struct {
	*utils.GenericClient
}

const (
	HeartbeatCheckerTimeout = 60
)

func NewDeployerClient(addr string) *Client {
	return &Client{
		GenericClient: utils.NewGenericClient(addr),
	}
}

func (c *Client) ExpandTree(serviceId, location string) (status int) {
	var reqBody api.ExpandTreeRequestBody
	reqBody = location

	path := deployer.GetExpandTreePath(serviceId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) GetServices() (serviceIds []string, status int) {
	path := deployer.GetDeploymentsPath()
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	var resp api.GetDeploymentsResponseBody
	status, _ = utils.DoRequest(c.Client, req, &resp)
	serviceIds = resp

	return
}

func (c *Client) RegisterService(serviceId string, static bool,
	deploymentYamlBytes []byte) (status int) {
	reqBody := api.RegisterServiceRequestBody{
		DeploymentId:        serviceId,
		Static:              static,
		DeploymentYAMLBytes: deploymentYamlBytes,
	}
	path := deployer.GetServicePath(serviceId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) ExtendDeploymentTo(serviceId, targetId string) (status int) {
	path := deployer.GetExtendServicePath(serviceId, targetId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) ShortenDeploymentFrom(serviceId, targetId string) (status int) {
	path := deployer.GetShortenServicePath(serviceId, targetId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) DeleteService(serviceId string) (status int) {
	path := deployer.GetServicePath(serviceId)
	req := utils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) RegisterServiceInstance(serviceId, instanceId string, static bool,
	portTranslation nat.PortMap, local bool) (status int) {
	reqBody := api.RegisterServiceInstanceRequestBody{
		Static:          static,
		PortTranslation: portTranslation,
		Local:           local,
	}
	path := deployer.GetServiceInstancePath(serviceId, instanceId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) RegisterHearbeatServiceInstance(serviceId, instanceId string) (status int) {
	path := deployer.GetServiceInstanceAlivePath(serviceId, instanceId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) SendHearbeatServiceInstance(serviceId, instanceId string) (status int) {
	path := deployer.GetServiceInstanceAlivePath(serviceId, instanceId)
	req := utils.BuildRequest(http.MethodPut, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) WarnOfDeadChild(serviceId, deadChildId string, grandChild *utils.Node) (status int) {
	var reqBody api.DeadChildRequestBody
	reqBody = *grandChild

	path := deployer.GetDeadChildPath(serviceId, deadChildId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) WarnToTakeChild(serviceId string, child *utils.Node) (status int) {
	var reqBody api.TakeChildRequestBody
	reqBody = *child

	path := deployer.GetDeploymentChildPath(serviceId, child.Id)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) WarnThatIAmParent(serviceId string, parent *utils.Node) (status int) {
	var reqBody api.IAmYourParentRequestBody
	reqBody = *parent

	path := deployer.GetImYourParentPath(serviceId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) ChildDeletedDeployment(serviceId, childId string) (status int) {
	path := deployer.GetDeploymentChildPath(serviceId, childId)
	req := utils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) MigrateDeployment(serviceId, origin, target string) (status int) {
	path := deployer.GetMigrateDeploymentPath(serviceId)
	reqBody := MigrateDTO{
		Origin: origin,
		Target: target,
	}

	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)
	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) GetHierarchyTable() (table map[string]*HierarchyEntryDTO, status int) {
	path := deployer.GetHierarchyTablePath()
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	var resp api.GetHierarchyTableResponseBody
	status, _ = utils.DoRequest(c.Client, req, &resp)

	table = resp

	return
}

func (c *Client) SetParentAlive(parentId string) (status int) {
	path := deployer.GetParentAlivePath(parentId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) AddNode(nodeAddr string) (status int) {
	var reqBody api.AddNodeRequestBody
	reqBody = nodeAddr

	path := deployer.GetAddNodePath()
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) SendInstanceHeartbeatToDeployerPeriodically() {
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
