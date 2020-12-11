package client

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	internalUtils "github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils/client"
	"github.com/docker/go-connections/nat"
	"github.com/golang/geo/s2"
	log "github.com/sirupsen/logrus"
)

type Client struct {
	utils.GenericClient
}

func NewArchimedesClient(addr string) *Client {
	newClient := client.NewGenericClient(addr)
	archClient := &Client{
		GenericClient: newClient,
	}

	newClient.Client.CheckRedirect = archClient.handleRedirect

	return archClient
}

func (c *Client) RegisterDeployment(deploymentId string, ports nat.PortSet, host *utils.Node) (status int) {
	reqBody := api.RegisterDeploymentRequestBody{
		Deployment: &api.DeploymentDTO{Ports: ports},
		Host:       host,
	}

	path := api.GetDeploymentPath(deploymentId)
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

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
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) DeleteDeployment(deploymentId string) (status int) {
	path := api.GetDeploymentPath(deploymentId)
	req := internalUtils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) DeleteDeploymentInstance(deploymentId, instanceId string) (status int) {
	path := api.GetDeploymentInstancePath(deploymentId, instanceId)
	req := internalUtils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) GetDeployments() (deployments map[string]*api.Deployment, status int) {
	req := internalUtils.BuildRequest(http.MethodGet, c.GetHostPort(), api.GetDeploymentsPath(), nil)

	deployments = api.GetAllDeploymentsResponseBody{}
	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, &deployments)
	return
}

func (c *Client) GetDeployment(deploymentId string) (instances map[string]*api.Instance, status int) {
	req := internalUtils.BuildRequest(http.MethodGet, c.GetHostPort(), api.GetDeploymentPath(deploymentId), nil)

	instances = api.GetDeploymentResponseBody{}
	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, &instances)
	return
}

func (c *Client) GetDeploymentInstance(deploymentId, instanceId string) (instance *api.Instance, status int) {
	path := api.GetDeploymentInstancePath(deploymentId, instanceId)
	req := internalUtils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	instance = &api.GetDeploymentInstanceResponseBody{}
	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, instance)

	return
}

func (c *Client) GetInstance(instanceId string) (instance *api.Instance, status int) {
	path := api.GetInstancePath(instanceId)
	req := internalUtils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	instance = &api.GetInstanceResponseBody{}
	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, instance)

	return
}

func (c *Client) Resolve(host string, port nat.Port, deploymentId string, cLocation s2.CellID,
	reqId string) (rHost, rPort string, status int, timedOut bool) {

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
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	var resp api.ResolveResponseBody
	status, timedOut = internalUtils.DoRequest(c.GetHTTPClient(), req, &resp)
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
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)
	return
}

func (c *Client) RemoveRedirect(deploymentId string) (status int) {
	path := api.GetRedirectPath(deploymentId)
	req := internalUtils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)
	return
}

func (c *Client) GetRedirected(deploymentId string) (redirected int32, status int) {
	path := api.GetRedirectedPath(deploymentId)
	req := internalUtils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, &redirected)
	return
}

func (c *Client) SetResolvingAnswer(id string, resolved *api.ResolvedDTO) (status int) {
	reqBody := api.SetResolutionAnswerRequestBody{
		Resolved: resolved,
		Id:       id,
	}

	path := api.SetResolvingAnswerPath
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)
	return
}

func (c *Client) GetLoad(deploymentId string) (load int, status int) {
	path := api.GetLoadPath(deploymentId)
	req := internalUtils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, &load)

	return
}

func (c *Client) GetClientCentroids(deploymentId string) (centroids []s2.CellID, status int) {
	path := api.GetAvgClientLocationPath(deploymentId)
	req := internalUtils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, &centroids)
	return
}

func (c *Client) SetExploringCells(deploymentId string, cells []s2.CellID) (status int) {
	var reqBody api.SetExploringClientLocationRequestBody
	reqBody = cells

	path := api.GetSetExploringClientLocationPath(deploymentId)
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) AddDeploymentNode(deploymentId string, node *utils.Node, location s2.CellID,
	exploring bool) (status int) {
	reqBody := api.AddDeploymentNodeRequestBody{
		Node:      node,
		Location:  location,
		Exploring: exploring,
	}

	path := api.GetAddDeploymentNodePath(deploymentId)
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) DeleteDeploymentNode(deploymentId string, nodeId string) (status int) {
	path := api.GetRemoveDeploymentNodePath(deploymentId, nodeId)
	req := internalUtils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) CanRedirectToYou(deploymentId, nodeId string) (can bool, status int) {
	path := api.GetRedirectingToYouPath(deploymentId, nodeId)
	req := internalUtils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)
	can = status == http.StatusOK

	return
}

func (c *Client) WillRedirectToYou(deploymentId, nodeId string) (status int) {
	path := api.GetRedirectingToYouPath(deploymentId, nodeId)
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) StopRedirectingToYou(deploymentId, nodeId string) (status int) {
	path := api.GetRedirectingToYouPath(deploymentId, nodeId)
	req := internalUtils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) handleRedirect(req *http.Request, via []*http.Request) error {
	log.Debugf("redirecting %s to %s", via[len(via)-1].URL.Host, req.URL.Host)

	if req.URL.Path == "/archimedes/resolve" {
		reqBody := api.ResolveRequestBody{}

		err := json.NewDecoder(req.Body).Decode(&reqBody)
		if err != nil {
			panic(err)
		}

		host, _, err := net.SplitHostPort(via[len(via)-1].URL.Host)
		if err != nil {
			panic(err)
		}

		reqBody.Redirects = append(reqBody.Redirects, host)
		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			panic(err)
		}

		req.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		req.ContentLength = int64(len(bodyBytes))
	}

	c.SetHostPort(req.URL.Host)
	return nil
}
