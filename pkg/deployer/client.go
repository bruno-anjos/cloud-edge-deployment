package deployer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	publicUtils "github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
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

func (c *Client) GetServices() (serviceIds []string, status int) {
	path := api.GetDeploymentsPath()
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	var resp api.GetDeploymentsResponseBody
	status, _ = utils.DoRequest(c.Client, req, &resp)
	serviceIds = resp

	return
}

func (c *Client) RegisterService(serviceId string, static bool,
	deploymentYamlBytes []byte, parent, grandparent *utils.Node) (status int) {
	reqBody := api.RegisterServiceRequestBody{
		Parent:              parent,
		Grandparent:         grandparent,
		DeploymentId:        serviceId,
		Static:              static,
		DeploymentYAMLBytes: deploymentYamlBytes,
	}
	path := api.GetDeploymentsPath()
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) ExtendDeploymentTo(serviceId, targetId string) (status int) {
	path := api.GetExtendServicePath(serviceId, targetId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) ShortenDeploymentFrom(serviceId, targetId string) (status int) {
	path := api.GetShortenServicePath(serviceId, targetId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) DeleteService(serviceId string) (status int) {
	path := api.GetServicePath(serviceId)
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
	path := api.GetServiceInstancePath(serviceId, instanceId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) RegisterHearbeatServiceInstance(serviceId, instanceId string) (status int) {
	path := api.GetServiceInstanceAlivePath(serviceId, instanceId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) SendHearbeatServiceInstance(serviceId, instanceId string) (status int) {
	path := api.GetServiceInstanceAlivePath(serviceId, instanceId)
	req := utils.BuildRequest(http.MethodPut, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) WarnOfDeadChild(serviceId, deadChildId string, grandChild *utils.Node,
	alternatives map[string]*utils.Node, location *publicUtils.Location) (status int) {
	var reqBody api.DeadChildRequestBody
	reqBody.Grandchild = grandChild
	reqBody.Alternatives = alternatives
	reqBody.Location = location

	path := api.GetDeadChildPath(serviceId, deadChildId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) SetGrandparent(serviceId string, grandparent *utils.Node) (status int) {
	var reqBody api.SetGrandparentRequestBody
	reqBody = *grandparent

	path := api.GetSetGrandparentPath(serviceId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) WarnToTakeChild(serviceId string, child *utils.Node) (status int) {
	var reqBody api.TakeChildRequestBody
	reqBody = *child

	path := api.GetDeploymentChildPath(serviceId, child.Id)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) WarnThatIAmParent(serviceId string, parent, grandparent *utils.Node) (status int) {
	reqBody := api.IAmYourParentRequestBody{}
	reqBody = append(reqBody, parent)
	reqBody = append(reqBody, grandparent)

	path := api.GetImYourParentPath(serviceId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) AskCanTakeChild(serviceId string, childId string) (status int) {
	path := api.GetCanTakeChildPath(serviceId, childId)
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) AskCanTakeParent(serviceId string, parentId string) (status int) {
	path := api.GetCanTakeParentPath(serviceId, parentId)
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) ChildDeletedDeployment(serviceId, childId string) (status int) {
	path := api.GetDeploymentChildPath(serviceId, childId)
	req := utils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) MigrateDeployment(serviceId, origin, target string) (status int) {
	path := api.GetMigrateDeploymentPath(serviceId)
	reqBody := api.MigrateDTO{
		Origin: origin,
		Target: target,
	}

	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)
	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) GetHierarchyTable() (table map[string]*api.HierarchyEntryDTO, status int) {
	path := api.GetHierarchyTablePath()
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	var resp api.GetHierarchyTableResponseBody
	status, _ = utils.DoRequest(c.Client, req, &resp)

	table = resp

	return
}

func (c *Client) SetParentAlive(parentId string) (status int) {
	path := api.GetParentAlivePath(parentId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) AddNode(nodeAddr string) (status int) {
	var reqBody api.AddNodeRequestBody
	reqBody = nodeAddr

	path := api.GetAddNodePath()
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

func (c *Client) SendAlternatives(myId string, alternatives []*utils.Node) (status int) {
	var reqBody api.AlternativesRequestBody
	reqBody = alternatives

	path := api.GetSetAlternativesPath(myId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) Fallback(deploymentId, orphanId string, orphanLocation *publicUtils.Location) (status int) {
	var reqBody api.FallbackRequestBody
	reqBody.OrphanId = orphanId
	reqBody.OrphanLocation = orphanLocation

	path := api.GetFallbackPath(deploymentId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) StartResolveUpTheTree(deploymentId string, toResolve *archimedes.ToResolveDTO) (status int) {
	var reqBody api.StartResolveUpTheTreeRequestBody
	reqBody = *toResolve
	path := api.GetStartResolveUpTheTreePath(deploymentId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) ResolveUpTheTree(deploymentId, origin string, toResolve *archimedes.ToResolveDTO) (status int) {
	reqBody := api.ResolveUpTheTreeRequestBody{
		Origin:    origin,
		ToResolve: toResolve,
	}

	path := api.GetResolveUpTheTreePath(deploymentId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) RedirectDownTheTree(deploymentId string, location *publicUtils.Location) (redirectTo string, status int) {
	var reqBody api.RedirectClientDownTheTreeRequestBody
	reqBody = location

	path := api.GetRedirectDownTheTreePath(deploymentId)
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, reqBody)

	var (
		resp     *http.Response
		respBody api.RedirectClientDownTheTreeResponseBody
	)
	status, resp = utils.DoRequest(c.Client, req, nil)
	if status == http.StatusOK {
		err := json.NewDecoder(resp.Body).Decode(&respBody)
		if err != nil {
			panic(err)
		}
		redirectTo = respBody
	}

	return
}

func (c *Client) GetFallback() (fallback string, status int) {
	path := api.GetGetFallbackIdPath()
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	var (
		respBody api.GetFallbackResponseBody
	)
	status, _ = utils.DoRequest(c.Client, req, &respBody)

	fallback = respBody

	return
}

func (c *Client) HasService(serviceId string) (has bool, status int) {
	path := api.GetHasDeploymentPath(serviceId)
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	has = status == http.StatusOK
	return
}

func (c *Client) SetTerminalLocations(deploymentId, origin string, locations ...*publicUtils.Location) (status int) {
	reqBody := api.TerminalLocationRequestBody{
		Child:     origin,
		Locations: locations,
	}

	path := api.GetTerminalLocationPath(deploymentId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) SetExploring(deploymentId, childId string) (status int) {
	path := api.GetSetExploringPath(deploymentId, childId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}
