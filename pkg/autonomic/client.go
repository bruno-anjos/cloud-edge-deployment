package autonomic

import (
	"net/http"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

type AutonomicClient struct {
	*utils.GenericClient
}

func NewAutonomicClient(addr string) *AutonomicClient {
	return &AutonomicClient{
		GenericClient: utils.NewGenericClient(addr, Port),
	}
}

func (c *AutonomicClient) RegisterService(serviceId, strategyId string) (status int) {
	reqBody := api.AddServiceRequestBody{
		StrategyId: strategyId,
	}

	path := autonomic.GetServicePath(serviceId)
	req := utils.BuildRequest(http.MethodPost, c.HostPort, path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *AutonomicClient) DeleteService(serviceId string) (status int) {
	path := autonomic.GetServicePath(serviceId)
	req := utils.BuildRequest(http.MethodDelete, c.HostPort, path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *AutonomicClient) GetServices() (services map[string]*AutonomicService, status int) {
	req := utils.BuildRequest(http.MethodGet, c.HostPort, autonomic.GetServicesPath(), nil)

	services = api.GetAllServicesResponseBody{}
	status, _ = utils.DoRequest(c.Client, req, &services)
	return
}
