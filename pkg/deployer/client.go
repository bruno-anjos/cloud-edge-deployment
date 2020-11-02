package deployer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/golang/geo/s2"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
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

func (c *Client) GetDeployments() (deploymentIds []string, status int) {
	path := api.GetDeploymentsPath()
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	var resp api.GetDeploymentsResponseBody
	status, _ = utils.DoRequest(c.Client, req, &resp)
	deploymentIds = resp

	return
}

func (c *Client) RegisterDeployment(deploymentId string, static bool,
	deploymentYamlBytes []byte, parent *utils.Node, children []*utils.Node) (status int) {
	reqBody := api.RegisterDeploymentRequestBody{
		Children:            children,
		Parent:              parent,
		DeploymentId:        deploymentId,
		Static:              static,
		DeploymentYAMLBytes: deploymentYamlBytes,
	}
	path := api.GetDeploymentsPath()
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) ExtendDeploymentTo(deploymentId, targetId string, parent *utils.Node, locations []s2.CellID,
	children []*utils.Node, exploring bool) (status int) {
	reqBody := api.ExtendDeploymentRequestBody{
		Parent:    parent,
		Children:  children,
		Exploring: exploring,
		Locations: locations,
	}

	path := api.GetExtendDeploymentPath(deploymentId, targetId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) ShortenDeploymentFrom(deploymentId, targetId string) (status int) {
	path := api.GetShortenDeploymentPath(deploymentId, targetId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) DeleteDeployment(deploymentId string) (status int) {
	path := api.GetDeploymentPath(deploymentId)
	req := utils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) RegisterDeploymentInstance(deploymentId, instanceId string, static bool,
	portTranslation nat.PortMap, local bool) (status int) {
	reqBody := api.RegisterDeploymentInstanceRequestBody{
		Static:          static,
		PortTranslation: portTranslation,
		Local:           local,
	}
	path := api.GetDeploymentInstancePath(deploymentId, instanceId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) RegisterHearbeatDeploymentInstance(deploymentId, instanceId string) (status int) {
	path := api.GetDeploymentInstanceAlivePath(deploymentId, instanceId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) SendHearbeatDeploymentInstance(deploymentId, instanceId string) (status int) {
	path := api.GetDeploymentInstanceAlivePath(deploymentId, instanceId)
	req := utils.BuildRequest(http.MethodPut, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) WarnOfDeadChild(deploymentId, deadChildId string, grandChild *utils.Node,
	alternatives map[string]*utils.Node, locations []s2.CellID) (status int) {
	var reqBody api.DeadChildRequestBody
	reqBody.Grandchild = grandChild
	reqBody.Alternatives = alternatives
	reqBody.Locations = locations

	path := api.GetDeadChildPath(deploymentId, deadChildId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) SetGrandparent(deploymentId string, grandparent *utils.Node) (status int) {
	var reqBody api.SetGrandparentRequestBody
	reqBody = *grandparent

	path := api.GetSetGrandparentPath(deploymentId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) WarnThatIAmParent(deploymentId string, parent, grandparent *utils.Node) (status int) {
	reqBody := api.IAmYourParentRequestBody{}
	reqBody = append(reqBody, parent)
	reqBody = append(reqBody, grandparent)

	path := api.GetImYourParentPath(deploymentId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) WarnThatIAmChild(deploymentId string, child *utils.Node) (grandparent *utils.Node, status int) {
	reqBody := api.IAmYourChildRequestBody{
		Child: child,
	}

	path := api.GetIAmYourChildPath(deploymentId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	var (
		resp *http.Response
	)
	status, resp = utils.DoRequest(c.Client, req, nil)
	if status == http.StatusOK {
		var respBody api.IAmYourChildResponseBody
		err := json.NewDecoder(resp.Body).Decode(&respBody)
		if err != nil {
			panic(err)
		}
		grandparent = &respBody
	}

	return
}

func (c *Client) ChildDeletedDeployment(deploymentId, childId string) (status int) {
	path := api.GetDeploymentChildPath(deploymentId, childId)
	req := utils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) MigrateDeployment(deploymentId, origin, target string) (status int) {
	path := api.GetMigrateDeploymentPath(deploymentId)
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
	deploymentId := os.Getenv(utils.DeploymentEnvVarName)
	instanceId := os.Getenv(utils.InstanceEnvVarName)

	status := c.RegisterHearbeatDeploymentInstance(deploymentId, instanceId)
	switch status {
	case http.StatusConflict:
		log.Debugf("deployment %s instance %s already has a heartbeat sender", deploymentId, instanceId)
		return
	case http.StatusOK:
	default:
		panic(errors.New(fmt.Sprintf("received unexpected status %d", status)))
	}

	ticker := time.NewTicker((HeartbeatCheckerTimeout / 3) * time.Second)
	for {
		<-ticker.C
		log.Info("sending heartbeat to deployer")
		status = c.SendHearbeatDeploymentInstance(deploymentId, instanceId)
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

func (c *Client) Fallback(deploymentId, orphanId string, orphanLocation s2.CellID) (status int) {
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

func (c *Client) RedirectDownTheTree(deploymentId string, location s2.CellID) (redirectTo string, status int) {
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

func (c *Client) HasDeployment(deploymentId string) (has bool, status int) {
	path := api.GetHasDeploymentPath(deploymentId)
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	has = status == http.StatusOK
	return
}

func (c *Client) PropagateLocationToHorizon(deploymentId, origin string, location s2.CellID,
	TTL int8) (status int) {
	reqBody := api.PropagateLocationToHorizonRequestBody{
		TTL:      TTL,
		ChildId:  origin,
		Location: location,
	}

	path := api.GetPropagateLocationToHorizonPath(deploymentId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}
