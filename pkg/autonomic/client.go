package autonomic

import (
	"encoding/json"
	"net/http"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
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

func (c *Client) RegisterDeployment(deploymentId, strategyId string) (status int) {
	reqBody := api.AddDeploymentRequestBody{
		StrategyId: strategyId,
	}

	path := api.GetDeploymentPath(deploymentId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) DeleteDeployment(deploymentId string) (status int) {
	path := api.GetDeploymentPath(deploymentId)
	req := utils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) GetDeployments() (deployments map[string]*api.DeploymentDTO, status int) {
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), api.GetDeploymentsPath(), nil)

	deployments = api.GetAllDeploymentsResponseBody{}
	status, _ = utils.DoRequest(c.Client, req, &deployments)
	return
}

func (c *Client) AddDeploymentChild(deploymentId, childId string) (status int) {
	path := api.GetDeploymentChildPath(deploymentId, childId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) RemoveDeploymentChild(deploymentId, childId string) (status int) {
	path := api.GetDeploymentChildPath(deploymentId, childId)
	req := utils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) SetDeploymentParent(deploymentId, parentId string) (status int) {
	path := api.GetDeploymentParentPath(deploymentId, parentId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) IsNodeInVicinity(nodeId string) (isInVicinity bool) {
	path := api.GetIsNodeInVicinityPath(nodeId)
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	status, _ := utils.DoRequest(c.Client, req, nil)
	if status == http.StatusOK {
		isInVicinity = true
	} else if status == http.StatusNotFound {
		isInVicinity = false
	} else {
		return false
	}

	return
}

func (c *Client) GetClosestNode(location s2.CellID, toExclude map[string]interface{}) (closest string) {
	reqBody := api.ClosestNodeRequestBody{
		Location:  location,
		ToExclude: toExclude,
	}
	path := api.GetClosestNodePath()
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, reqBody)

	var respBody api.ClosestNodeResponseBody
	status, resp := utils.DoRequest(c.Client, req, nil)
	if status == http.StatusOK {
		err := json.NewDecoder(resp.Body).Decode(&respBody)
		if err != nil {
			panic(err)
		}
		closest = respBody
	} else {
		closest = ""
	}

	return
}

func (c *Client) GetVicinity() (vicinity map[string]s2.CellID, status int) {
	path := api.GetVicinityPath()
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	var (
		respBody api.GetVicinityResponseBody
		resp     *http.Response
	)
	status, resp = utils.DoRequest(c.Client, req, nil)
	if status == http.StatusOK {
		err := json.NewDecoder(resp.Body).Decode(&respBody)
		if err != nil {
			panic(err)
		}
		vicinity = respBody
	} else {
		vicinity = nil
	}

	return
}

func (c *Client) GetLocation() (location s2.CellID, status int) {
	path := api.GetMyLocationPath()
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	var (
		respBody api.GetMyLocationResponseBody
		resp     *http.Response
	)
	status, resp = utils.DoRequest(c.Client, req, nil)
	if status == http.StatusOK {
		err := json.NewDecoder(resp.Body).Decode(&respBody)
		if err != nil {
			panic(err)
		}
		location = respBody
	} else {
		location = 0
	}

	return
}

func (c *Client) SetExploredSuccessfully(deploymentId, childId string) (status int) {
	path := api.GetExploredPath(deploymentId, childId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) BlacklistNode(deploymentId, nodeId string) (status int) {
	path := api.GetBlacklistPath(deploymentId, nodeId)

	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, nil)
	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}
