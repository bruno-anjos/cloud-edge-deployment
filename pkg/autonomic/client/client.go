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

func (c *Client) RegisterDeployment(deploymentID, strategyID string, depthFactor float64,
	exploringTTL int) (status int) {
	reqBody := api.AddDeploymentRequestBody{
		StrategyID:   strategyID,
		ExploringTTL: exploringTTL,
		DepthFactor:  depthFactor,
	}

	path := api.GetDeploymentPath(deploymentID)
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) DeleteDeployment(deploymentID string) (status int) {
	path := api.GetDeploymentPath(deploymentID)
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

func (c *Client) AddDeploymentChild(deploymentID string, child *utils.Node) (status int) {
	path := api.GetDeploymentChildPath(deploymentID)

	reqBody := *child

	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) RemoveDeploymentChild(deploymentID, childID string) (status int) {
	path := api.GetDeploymentChildWithChildPath(deploymentID, childID)
	req := internalUtils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) SetDeploymentParent(deploymentID string, parent *utils.Node) (status int) {
	path := api.GetDeploymentParentPath(deploymentID)

	reqBody := *parent
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) IsNodeInVicinity(nodeID string) (isInVicinity bool) {
	path := api.GetIsNodeInVicinityPath(nodeID)
	req := internalUtils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	status, _ := internalUtils.DoRequest(c.GetHTTPClient(), req, nil)
	switch status {
	case http.StatusOK:
		isInVicinity = true
	case http.StatusNotFound:
		isInVicinity = false
	default:
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

func (c *Client) SetExploredSuccessfully(deploymentID, childID string) (status int) {
	path := api.GetExploredPath(deploymentID, childID)
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) BlacklistNodes(deploymentID, origin string, nodes ...string) (status int) {
	path := api.GetBlacklistPath(deploymentID)
	reqBody := api.BlacklistNodeRequestBody{Origin: origin, Nodes: nodes}

	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)
	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}
