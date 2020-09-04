package archimedes

import (
	"net"
	"net/http"
	"strconv"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/docker/go-connections/nat"
	log "github.com/sirupsen/logrus"
)

type Client struct {
	*utils.GenericClient
}

func NewArchimedesClient(addr string) *Client {
	newClient := utils.NewGenericClient(addr, Port)
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

	path := archimedes.GetServicePath(serviceId)
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

	path := archimedes.GetServiceInstancePath(serviceId, instanceId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) DeleteService(serviceId string) (status int) {
	path := archimedes.GetServicePath(serviceId)
	req := utils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) DeleteServiceInstance(serviceId, instanceId string) (status int) {
	path := archimedes.GetServiceInstancePath(serviceId, instanceId)
	req := utils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) GetServices() (services map[string]*Service, status int) {
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), archimedes.GetServicesPath(), nil)

	services = api.GetAllServicesResponseBody{}
	status, _ = utils.DoRequest(c.Client, req, &services)
	return
}

func (c *Client) GetService(serviceId string) (instances map[string]*Instance, status int) {
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), archimedes.GetServicePath(serviceId), nil)

	instances = api.GetServiceResponseBody{}
	status, _ = utils.DoRequest(c.Client, req, &instances)
	return
}

func (c *Client) GetServiceInstance(serviceId, instanceId string) (instance *Instance, status int) {
	path := archimedes.GetServiceInstancePath(serviceId, instanceId)
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	instance = &api.GetServiceInstanceResponseBody{}
	status, _ = utils.DoRequest(c.Client, req, instance)

	return
}

func (c *Client) GetInstance(instanceId string) (instance *Instance, status int) {
	path := archimedes.GetInstancePath(instanceId)
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	instance = &api.GetInstanceResponseBody{}
	status, _ = utils.DoRequest(c.Client, req, instance)

	return
}

func (c *Client) Resolve(host string, port nat.Port) (rHost, rPort string, status int) {
	reqBody := api.ResolveRequestBody{
		Host: host,
		Port: port,
	}

	path := archimedes.GetResolvePath()
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

	path := archimedes.GetRedirectPath(serviceId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)
	return
}

func (c *Client) RemoveRedirect(serviceId string) (status int) {
	path := archimedes.GetRedirectPath(serviceId)
	req := utils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)
	return
}

func (c *Client) GetRedirected(serviceId string) (redirected int32, status int) {
	path := archimedes.GetRedirectedPath(serviceId)
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, redirected)
	return
}

func (c *Client) handleRedirect(req *http.Request, via []*http.Request) error {
	host, portString, err := net.SplitHostPort(req.URL.Host)
	if err != nil {
		log.Error("error handling redirect: ", err)
		return err
	}

	port, err := strconv.Atoi(portString)
	if err != nil {
		log.Error("error converting string port to int: ", err)
		return err
	}

	log.Debugf("redirecting %s to %s", via[len(via)-1].URL.Host, req.URL.Host)

	c.SetHostPort(host, port)
	return nil
}
