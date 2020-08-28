package archimedes

import (
	"net/http"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/docker/go-connections/nat"
)

type ArchimedesClient struct {
	*utils.GenericClient
}

func NewArchimedesClient(addr string) *ArchimedesClient {
	return &ArchimedesClient{
		GenericClient: utils.NewGenericClient(addr, archimedes.Port),
	}
}

func (c *ArchimedesClient) RegisterService(serviceId string, ports nat.PortSet) (status int) {
	reqBody := api.RegisterServiceRequestBody{
		Ports: ports,
	}

	path := archimedes.GetServicePath(serviceId)
	req := utils.BuildRequest(http.MethodPost, c.HostPort, path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *ArchimedesClient) RegisterServiceInstance(serviceId, instanceId string, static bool,
	portTranslation nat.PortMap, local bool) (status int) {
	reqBody := api.RegisterServiceInstanceRequestBody{
		Static:          static,
		PortTranslation: portTranslation,
		Local:           local,
	}

	path := archimedes.GetServiceInstancePath(serviceId, instanceId)
	req := utils.BuildRequest(http.MethodPost, c.HostPort, path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *ArchimedesClient) DeleteService(serviceId string) (status int) {
	path := archimedes.GetServicePath(serviceId)
	req := utils.BuildRequest(http.MethodDelete, c.HostPort, path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *ArchimedesClient) DeleteServiceInstance(serviceId, instanceId string) (status int) {
	path := archimedes.GetServiceInstancePath(serviceId, instanceId)
	req := utils.BuildRequest(http.MethodDelete, c.HostPort, path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *ArchimedesClient) GetServices() (services map[string]*Service, status int) {
	req := utils.BuildRequest(http.MethodGet, c.HostPort, archimedes.GetServicesPath(), nil)

	services = api.GetAllServicesReponseBody{}
	status, _ = utils.DoRequest(c.Client, req, &services)
	return
}

func (c *ArchimedesClient) GetService(serviceId string) (instances map[string]*Instance, status int) {
	req := utils.BuildRequest(http.MethodGet, c.HostPort, archimedes.GetServicePath(serviceId), nil)

	instances = api.GetServiceResponseBody{}
	status, _ = utils.DoRequest(c.Client, req, &instances)
	return
}

func (c *ArchimedesClient) GetServiceInstance(serviceId, instanceId string) (instance *Instance, status int) {
	path := archimedes.GetServiceInstancePath(serviceId, instanceId)
	req := utils.BuildRequest(http.MethodGet, c.HostPort, path, nil)

	instance = &api.GetServiceInstanceResponseBody{}
	status, _ = utils.DoRequest(c.Client, req, instance)

	return
}

func (c *ArchimedesClient) GetInstance(instanceId string) (instance *Instance, status int) {
	path := archimedes.GetInstancePath(instanceId)
	req := utils.BuildRequest(http.MethodGet, c.HostPort, path, nil)

	instance = &api.GetInstanceResponseBody{}
	status, _ = utils.DoRequest(c.Client, req, instance)

	return
}

func (c *ArchimedesClient) Resolve(host string, port nat.Port) (rHost, rPort string, status int) {
	reqBody := api.ResolveRequestBody{
		Host: host,
		Port: port,
	}

	path := archimedes.GetResolvePath()
	req := utils.BuildRequest(http.MethodPost, c.HostPort, path, reqBody)

	var resp api.ResolveResponseBody
	status, _ = utils.DoRequest(c.Client, req, &resp)

	rHost = resp.Host
	rPort = resp.Port

	return
}
