package archimedes

import (
	"encoding/json"
	"net/http"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/docker/go-connections/nat"
	"github.com/golang/geo/s2"
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

func (c *Client) RegisterDeployment(deploymentId string, ports nat.PortSet) (status int) {
	reqBody := api.RegisterDeploymentRequestBody{
		Ports: ports,
	}

	path := api.GetDeploymentPath(deploymentId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) RegisterDeploymentInstance(deploymentId, instanceId string, static bool,
	portTranslation nat.PortMap, local bool) (status int) {
	reqBody := api.RegisterDeploymentInstanceRequestBody{
		Static:          static,
		PortTranslation: portTranslation,
		Local:           local,
	}

	path := api.GetDeploymentInstancePath(deploymentId, instanceId)
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

func (c *Client) DeleteDeploymentInstance(deploymentId, instanceId string) (status int) {
	path := api.GetDeploymentInstancePath(deploymentId, instanceId)
	req := utils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) GetDeployments() (deployments map[string]*api.Deployment, status int) {
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), api.GetDeploymentsPath(), nil)

	deployments = api.GetAllDeploymentsResponseBody{}
	status, _ = utils.DoRequest(c.Client, req, &deployments)
	return
}

func (c *Client) GetDeployment(deploymentId string) (instances map[string]*api.Instance, status int) {
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), api.GetDeploymentPath(deploymentId), nil)

	instances = api.GetDeploymentResponseBody{}
	status, _ = utils.DoRequest(c.Client, req, &instances)
	return
}

func (c *Client) GetDeploymentInstance(deploymentId, instanceId string) (instance *api.Instance, status int) {
	path := api.GetDeploymentInstancePath(deploymentId, instanceId)
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	instance = &api.GetDeploymentInstanceResponseBody{}
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

func (c *Client) Resolve(host string, port nat.Port, deploymentId string, cLocation s2.CellID,
	reqId string) (rHost, rPort string, status int) {

	reqBody := api.ResolveRequestBody{
		ToResolve: &api.ToResolveDTO{
			Host: host,
			Port: port,
		},
		DeploymentId: deploymentId,
		Location:     cLocation,
		Id:           reqId,
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

func (c *Client) Redirect(deploymentId, target string, amount int) (status int) {
	reqBody := api.RedirectRequestBody{
		Amount: int32(amount),
		Target: target,
	}

	path := api.GetRedirectPath(deploymentId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)
	return
}

func (c *Client) RemoveRedirect(deploymentId string) (status int) {
	path := api.GetRedirectPath(deploymentId)
	req := utils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)
	return
}

func (c *Client) GetRedirected(deploymentId string) (redirected int32, status int) {
	path := api.GetRedirectedPath(deploymentId)
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	var resp *http.Response
	status, resp = utils.DoRequest(c.Client, req, nil)

	if status == http.StatusOK {
		err := json.NewDecoder(resp.Body).Decode(&redirected)
		if err != nil {
			panic(err)
		}
	}

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

func (c *Client) GetLoad(deploymentId string) (load int, status int) {
	path := api.GetLoadPath(deploymentId)
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, &load)

	return
}

func (c *Client) GetClientCentroids(deploymentId string) (centroids []s2.CellID, status int) {
	path := api.GetAvgClientLocationPath(deploymentId)
	req := utils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	var resp *http.Response
	status, resp = utils.DoRequest(c.Client, req, nil)

	if status == http.StatusOK {
		err := json.NewDecoder(resp.Body).Decode(&centroids)
		if err != nil {
			panic(err)
		}
	}

	return
}

func (c *Client) SetExploringCells(deploymentId string, cells []s2.CellID) (status int) {
	var reqBody api.SetExploringClientLocationRequestBody
	reqBody = cells

	path := api.GetSetExploringClientLocationPath(deploymentId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) AddDeploymentNode(deploymentId string, nodeId string, location s2.CellID,
	exploring bool) (status int) {
	reqBody := api.AddDeploymentNodeRequestBody{
		NodeId:    nodeId,
		Location:  location,
		Exploring: exploring,
	}

	path := api.GetAddDeploymentNodePath(deploymentId)
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) DeleteDeploymentNode(deploymentId string, nodeId string) (status int) {
	path := api.GetRemoveDeploymentNodePath(deploymentId, nodeId)
	req := utils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) handleRedirect(req *http.Request, via []*http.Request) error {
	log.Debugf("redirecting %s to %s", via[len(via)-1].URL.Host, req.URL.Host)

	c.SetHostPort(req.URL.Host)
	return nil
}
