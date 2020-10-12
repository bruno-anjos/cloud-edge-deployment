package autonomic

import (
	"encoding/json"
	"net/http"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	publicUtils "github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
)

type Client struct {
	*utils.GenericClient
}

func NewAutonomicClient(addr string) *Client {
	return &Client{
		GenericClient: utils.NewGenericClient(addr),
	}
}

func (c *Client) RegisterService(serviceId, strategyId string) (status int) {
	reqBody := api.AddServiceRequestBody{
		StrategyId: strategyId,
	}

	path := api.GetServicePath(serviceId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) DeleteService(serviceId string) (status int) {
	path := api.GetServicePath(serviceId)
	req := utils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) GetServices() (services map[string]*api.ServiceDTO, status int) {
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), api.GetServicesPath(), nil)

	services = api.GetAllServicesResponseBody{}
	status, _ = utils.DoRequest(c.Client, req, &services)
	return
}

func (c *Client) AddServiceChild(serviceId, childId string) (status int) {
	path := api.GetServiceChildPath(serviceId, childId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) RemoveServiceChild(serviceId, childId string) (status int) {
	path := api.GetServiceChildPath(serviceId, childId)
	req := utils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) SetServiceParent(serviceId, parentId string) (status int) {
	path := api.GetServiceParentPath(serviceId, parentId)
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

func (c *Client) GetClosestNode(location *publicUtils.Location, toExclude map[string]struct{}) (closest string) {
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

func (c *Client) GetVicinity() (vicinity map[string]interface{}, status int) {
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

func (c *Client) GetLocation() (location *publicUtils.Location, status int) {
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
		location = nil
	}

	return
}

func (c *Client) GetLoadForService(serviceId string) (load float64, status int) {
	path := api.GetGetLoadForServicePath(serviceId)
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	var (
		respBody api.GetLoadForServiceResponseBody
		resp     *http.Response
	)
	status, resp = utils.DoRequest(c.Client, req, nil)
	if status == http.StatusOK {
		err := json.NewDecoder(resp.Body).Decode(&respBody)
		if err != nil {
			panic(err)
		}
		load = respBody
	}

	return
}

func (c *Client) SetExploredSuccessfully(serviceId, childId string) (status int) {
	path := api.GetExploredPath(serviceId, childId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}
