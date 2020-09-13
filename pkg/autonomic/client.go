package autonomic

import (
	"net/http"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
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
