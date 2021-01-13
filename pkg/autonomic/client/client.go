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

func NewAutonomicClient() *Client {
	return &Client{
		GenericClient: client.NewGenericClient(),
	}
}

func (c *Client) GetID(addr string) (id string, status int) {
	path := api.GetGetIDPath()
	req := internalUtils.BuildRequest(http.MethodGet, addr, path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, &id)

	return
}

func (c *Client) RegisterDeployment(addr, deploymentID, strategyID string, depthFactor float64,
	exploringTTL int) (status int) {
	reqBody := api.AddDeploymentRequestBody{
		StrategyID:   strategyID,
		ExploringTTL: exploringTTL,
		DepthFactor:  depthFactor,
	}

	path := api.GetDeploymentPath(deploymentID)
	req := internalUtils.BuildRequest(http.MethodPost, addr, path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) DeleteDeployment(addr, deploymentID string) (status int) {
	path := api.GetDeploymentPath(deploymentID)
	req := internalUtils.BuildRequest(http.MethodDelete, addr, path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) GetDeployments(addr string) (deployments map[string]*api.DeploymentDTO, status int) {
	req := internalUtils.BuildRequest(http.MethodGet, addr, api.GetDeploymentsPath(), nil)

	deployments = api.GetAllDeploymentsResponseBody{}
	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, &deployments)

	return
}

func (c *Client) AddDeploymentChild(addr, deploymentID string, child *utils.Node) (status int) {
	path := api.GetDeploymentChildPath(deploymentID)

	reqBody := *child

	req := internalUtils.BuildRequest(http.MethodPost, addr, path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) RemoveDeploymentChild(addr, deploymentID, childID string) (status int) {
	path := api.GetDeploymentChildWithChildPath(deploymentID, childID)
	req := internalUtils.BuildRequest(http.MethodDelete, addr, path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) SetDeploymentParent(addr, deploymentID string, parent *utils.Node) (status int) {
	path := api.GetDeploymentParentPath(deploymentID)

	reqBody := *parent
	req := internalUtils.BuildRequest(http.MethodPost, addr, path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) IsNodeInVicinity(addr, nodeID string) (isInVicinity bool) {
	path := api.GetIsNodeInVicinityPath(nodeID)
	req := internalUtils.BuildRequest(http.MethodGet, addr, path, nil)

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

func (c *Client) GetClosestNode(addr string, locations []s2.CellID, toExclude map[string]interface{}) (closest *utils.
	Node) {
	reqBody := api.ClosestNodeRequestBody{
		Locations: locations,
		ToExclude: toExclude,
	}
	path := api.GetClosestNodePath()
	req := internalUtils.BuildRequest(http.MethodGet, addr, path, reqBody)

	respBody := api.ClosestNodeResponseBody{}
	internalUtils.DoRequest(c.GetHTTPClient(), req, &respBody)

	closest = &respBody

	return
}

func (c *Client) SetExploredSuccessfully(addr, deploymentID, childID string) (status int) {
	path := api.GetExploredPath(deploymentID, childID)
	req := internalUtils.BuildRequest(http.MethodPost, addr, path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) BlacklistNodes(addr, deploymentID, origin string, nodes []string,
	nodesVisited map[string]struct{}) (status int) {
	path := api.GetBlacklistPath(deploymentID)
	reqBody := api.BlacklistNodeRequestBody{Origin: origin, Nodes: nodes, NodesVisited: nodesVisited}

	req := internalUtils.BuildRequest(http.MethodPost, addr, path, reqBody)
	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}
