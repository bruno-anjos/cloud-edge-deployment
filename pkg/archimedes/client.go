package archimedes

import (
	"net/http"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/docker/go-connections/nat"
	log "github.com/sirupsen/logrus"
)

type Client struct {
	*utils.GenericClient
}

func NewArchimedesClient(addr string) *Client {
	newClient := utils.NewGenericClient(addr)
	archClient := &Client{
		GenericClient: newClient,
	}

	newClient.Client.CheckRedirect = archClient.handleRedirect

	return archClient
}

func (c *Client) RegisterService(serviceId string, ports nat.PortSet) (status int) {
	reqBody := api.RegisterServiceRequestBody{
		Ports: ports,
	}

	path := api.GetServicePath(serviceId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) RegisterServiceInstance(serviceId, instanceId string, static bool,
	portTranslation nat.PortMap, local bool) (status int) {
	reqBody := api.RegisterServiceInstanceRequestBody{
		Static:          static,
		PortTranslation: portTranslation,
		Local:           local,
	}

	path := api.GetServiceInstancePath(serviceId, instanceId)
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

func (c *Client) DeleteServiceInstance(serviceId, instanceId string) (status int) {
	path := api.GetServiceInstancePath(serviceId, instanceId)
	req := utils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) GetServices() (services map[string]*api.Service, status int) {
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), api.GetServicesPath(), nil)

	services = api.GetAllServicesResponseBody{}
	status, _ = utils.DoRequest(c.Client, req, &services)
	return
}

func (c *Client) GetService(serviceId string) (instances map[string]*api.Instance, status int) {
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), api.GetServicePath(serviceId), nil)

	instances = api.GetServiceResponseBody{}
	status, _ = utils.DoRequest(c.Client, req, &instances)
	return
}

func (c *Client) GetServiceInstance(serviceId, instanceId string) (instance *api.Instance, status int) {
	path := api.GetServiceInstancePath(serviceId, instanceId)
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	instance = &api.GetServiceInstanceResponseBody{}
	status, _ = utils.DoRequest(c.Client, req, instance)

	return
}

func (c *Client) GetInstance(instanceId string) (instance *api.Instance, status int) {
	path := api.GetInstancePath(instanceId)
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	instance = &api.GetInstanceResponseBody{}
	status, _ = utils.DoRequest(c.Client, req, instance)

	return
}

func (c *Client) Resolve(host string, port nat.Port, deploymentId string) (rHost, rPort string, status int) {
	reqBody := api.ResolveRequestBody{
		ToResolve: &api.ToResolveDTO{
			Host: host,
			Port: port,
		},
		DeploymentId: deploymentId,
	}

	path := api.GetResolvePath()
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	var resp api.ResolveResponseBody
	status, _ = utils.DoRequest(c.Client, req, &resp)
	rHost = resp.Host
	rPort = resp.Port

	return
}

func (c *Client) ResolveLocally(host string, port nat.Port) (rHost, rPort string, status int) {
	reqBody := api.ResolveLocallyRequestBody{
		Host: host,
		Port: port,
	}

	path := api.GetResolveLocallyPath()
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	var resp api.ResolveResponseBody
	status, _ = utils.DoRequest(c.Client, req, &resp)
	rHost = resp.Host
	rPort = resp.Port

	return
}

func (c *Client) Redirect(serviceId, target string, amount int) (status int) {
	reqBody := api.RedirectRequestBody{
		Amount: int32(amount),
		Target: target,
	}

	path := api.GetRedirectPath(serviceId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)
	return
}

func (c *Client) RemoveRedirect(serviceId string) (status int) {
	path := api.GetRedirectPath(serviceId)
	req := utils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)
	return
}

func (c *Client) GetRedirected(serviceId string) (redirected int32, status int) {
	path := api.GetRedirectedPath(serviceId)
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, redirected)
	return
}

func (c *Client) SetResolvingAnswer(id string, resolved *api.ResolvedDTO) (status int) {
	reqBody := api.SetResolutionAnswerRequestBody{
		Resolved: resolved,
		Id:       id,
	}

	path := api.SetResolvingAnswerPath
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)
	return
}

func (c *Client) handleRedirect(req *http.Request, via []*http.Request) error {
	log.Debugf("redirecting %s to %s", via[len(via)-1].URL.Host, req.URL.Host)

	c.SetHostPort(req.URL.Host)
	return nil
}
