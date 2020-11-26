package autonomic

import (
	"net/http"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/golang/geo/s2"
)

type Client struct {
	*utils.GenericClient
}

func NewAutonomicClient(addr string) *Client {
	return &Client{
		GenericClient: utils.NewGenericClient(addr),
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
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) DeleteDeployment(deploymentId string) (status int) {
	path := api.GetDeploymentPath(deploymentId)
	req := utils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) GetDeployments() (deployments map[string]*api.DeploymentDTO, status int) {
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), api.GetDeploymentsPath(), nil)

	deployments = api.GetAllDeploymentsResponseBody{}
	status = utils.DoRequest(c.Client, req, &deployments)
	return
}

func (c *Client) AddDeploymentChild(deploymentId, childId string) (status int) {
	path := api.GetDeploymentChildPath(deploymentId, childId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, nil)

	status = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) RemoveDeploymentChild(deploymentId, childId string) (status int) {
	path := api.GetDeploymentChildPath(deploymentId, childId)
	req := utils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) SetDeploymentParent(deploymentId, parentId string) (status int) {
	path := api.GetDeploymentParentPath(deploymentId, parentId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, nil)

	status = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) IsNodeInVicinity(nodeId string) (isInVicinity bool) {
	path := api.GetIsNodeInVicinityPath(nodeId)
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	status := utils.DoRequest(c.Client, req, nil)
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
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, reqBody)

	respBody := api.ClosestNodeResponseBody{}
	utils.DoRequest(c.Client, req, &respBody)

	closest = &respBody

	return
}

func (c *Client) GetVicinity() (vicinity *autonomic.Vicinity, status int) {
	path := api.GetVicinityPath()
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	respBody := api.GetVicinityResponseBody{}
	status = utils.DoRequest(c.Client, req, &respBody)

	vicinity = &respBody

	return
}

func (c *Client) GetLocation() (location s2.CellID, status int) {
	path := api.GetMyLocationPath()
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	status = utils.DoRequest(c.Client, req, &location)
	return
}

func (c *Client) SetExploredSuccessfully(deploymentId, childId string) (status int) {
	path := api.GetExploredPath(deploymentId, childId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, nil)

	status = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) BlacklistNodes(deploymentId, origin string, nodes ...string) (status int) {
	path := api.GetBlacklistPath(deploymentId)
	reqBody := api.BlacklistNodeRequestBody{Origin: origin, Nodes: nodes}

	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)
	status = utils.DoRequest(c.Client, req, nil)

	return
}
