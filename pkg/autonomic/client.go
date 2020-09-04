package autonomic

import (
	"net/http"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

type Client struct {
	*utils.GenericClient
}

func NewAutonomicClient(addr string) *Client {
	return &Client{
		GenericClient: utils.NewGenericClient(addr, Port),
	}
}

func (c *Client) RegisterService(serviceId, strategyId string) (status int) {
	reqBody := api.AddServiceRequestBody{
		StrategyId: strategyId,
	}

	path := autonomic.GetServicePath(serviceId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) DeleteService(serviceId string) (status int) {
	path := autonomic.GetServicePath(serviceId)
	req := utils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) GetServices() (services map[string]*Service, status int) {
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), autonomic.GetServicesPath(), nil)

	services = api.GetAllServicesResponseBody{}
	status, _ = utils.DoRequest(c.Client, req, &services)
	return
}

func (c *Client) AddServiceChild(serviceId, childId string) (status int) {
	path := autonomic.GetServiceChildPath(serviceId, childId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) RemoveServiceChild(serviceId, childId string) (status int) {
	path := autonomic.GetServiceChildPath(serviceId, childId)
	req := utils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}
