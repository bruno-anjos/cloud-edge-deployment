package client

import (
	"net/http"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
	internalUtils "github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils/client"

	"github.com/golang/geo/s2"
)

type Client struct {
	utils.GenericClient
}

func NewAutonomicClient(addr string) *Client {
	return &Client{
		GenericClient: client.NewGenericClient(addr),
	}
}

func (c *Client) RegisterDeployment(deploymentId, strategyId string, depthFactor float64,
	exploringTTL int) (status int) {
	reqBody := api.AddDeploymentRequestBody{
		StrategyId:   strategyId,
		ExploringTTL: exploringTTL,
		DepthFactor:  depthFactor,
	}

	path := api.GetDeploymentPath(deploymentId)
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) DeleteDeployment(deploymentId string) (status int) {
	path := api.GetDeploymentPath(deploymentId)
	req := internalUtils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) GetDeployments() (deployments map[string]*api.DeploymentDTO, status int) {
	req := internalUtils.BuildRequest(http.MethodGet, c.GetHostPort(), api.GetDeploymentsPath(), nil)

	deployments = api.GetAllDeploymentsResponseBody{}
	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, &deployments)
	return
}

func (c *Client) AddDeploymentChild(deploymentId string, child *utils.Node) (status int) {
	path := api.GetDeploymentChildPath(deploymentId)

	var reqBody api.AddDeploymentChildRequestBody
	reqBody = *child

	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) RemoveDeploymentChild(deploymentId, childId string) (status int) {
	path := api.GetDeploymentChildWithChildPath(deploymentId, childId)
	req := internalUtils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) SetDeploymentParent(deploymentId string, parent *utils.Node) (status int) {
	path := api.GetDeploymentParentPath(deploymentId)

	var reqBody api.SetDeploymentParentRequestBody
	reqBody = *parent
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) IsNodeInVicinity(nodeId string) (isInVicinity bool) {
	path := api.GetIsNodeInVicinityPath(nodeId)
	req := internalUtils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	status, _ := internalUtils.DoRequest(c.GetHTTPClient(), req, nil)
	if status == http.StatusOK {
		isInVicinity = true
	} else if status == http.StatusNotFound {
		isInVicinity = false
	} else {
		return false
	}

	return
}

func (c *Client) GetClosestNode(locations []s2.CellID, toExclude map[string]interface{}) (closest *utils.Node) {
	reqBody := api.ClosestNodeRequestBody{
		Locations: locations,
		ToExclude: toExclude,
	}
	path := api.GetClosestNodePath()
	req := internalUtils.BuildRequest(http.MethodGet, c.GetHostPort(), path, reqBody)

	respBody := api.ClosestNodeResponseBody{}
	internalUtils.DoRequest(c.GetHTTPClient(), req, &respBody)

	closest = &respBody

	return
}

func (c *Client) GetVicinity() (vicinity *api.Vicinity, status int) {
	path := api.GetVicinityPath()
	req := internalUtils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	respBody := api.GetVicinityResponseBody{}
	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, &respBody)

	vicinity = &respBody

	return
}

func (c *Client) GetLocation() (location s2.CellID, status int) {
	path := api.GetMyLocationPath()
	req := internalUtils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, &location)
	return
}

func (c *Client) SetExploredSuccessfully(deploymentId, childId string) (status int) {
	path := api.GetExploredPath(deploymentId, childId)
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) BlacklistNodes(deploymentId, origin string, nodes ...string) (status int) {
	path := api.GetBlacklistPath(deploymentId)
	reqBody := api.BlacklistNodeRequestBody{Origin: origin, Nodes: nodes}

	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)
	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}
