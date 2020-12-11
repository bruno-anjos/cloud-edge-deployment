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

func (c *Client) RegisterDeployment(deploymentID string, ports nat.PortSet, host *utils.Node) (status int) {
	reqBody := api.RegisterDeploymentRequestBody{
		Deployment: &api.DeploymentDTO{Ports: ports},
		Host:       host,
	}

	path := api.GetDeploymentPath(deploymentID)
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) RegisterDeploymentInstance(
	deploymentID, instanceID string, static bool, portTranslation nat.PortMap, local bool,
) (status int) {
	reqBody := api.RegisterDeploymentInstanceRequestBody{
		Static:          static,
		PortTranslation: portTranslation,
		Local:           local,
	}

	path := api.GetDeploymentInstancePath(deploymentID, instanceID)
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) DeleteDeployment(deploymentID string) (status int) {
	path := api.GetDeploymentPath(deploymentID)
	req := internalUtils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) DeleteDeploymentInstance(deploymentID, instanceID string) (status int) {
	path := api.GetDeploymentInstancePath(deploymentID, instanceID)
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

func (c *Client) GetDeployment(deploymentID string) (instances map[string]*api.Instance, status int) {
	req := internalUtils.BuildRequest(http.MethodGet, c.GetHostPort(), api.GetDeploymentPath(deploymentID), nil)

	instances = api.GetDeploymentResponseBody{}
	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, &instances)

	return
}

func (c *Client) GetDeploymentInstance(deploymentID, instanceID string) (instance *api.Instance, status int) {
	path := api.GetDeploymentInstancePath(deploymentID, instanceID)
	req := internalUtils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	instance = &api.GetDeploymentInstanceResponseBody{}
	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, instance)

	return
}

func (c *Client) GetInstance(instanceID string) (instance *api.Instance, status int) {
	path := api.GetInstancePath(instanceID)
	req := internalUtils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	instance = &api.GetInstanceResponseBody{}
	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, instance)

	return
}

func (c *Client) Resolve(host string, port nat.Port, deploymentID string, cLocation s2.CellID,
	reqID string) (rHost, rPort string, status int, timedOut bool) {
	reqBody := api.ResolveRequestBody{
		ToResolve: &api.ToResolveDTO{
			Host: host,
			Port: port,
		},
		DeploymentID: deploymentID,
		Location:     cLocation,
		ID:           reqID,
		Redirects:    nil,
	}

	path := api.GetResolvePath()
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	var resp api.ResolveResponseBody
	status, timedOut = internalUtils.DoRequest(c.GetHTTPClient(), req, &resp)
	rHost = resp.Host
	rPort = resp.Port

	return
}

func (c *Client) Redirect(deploymentID, target string, amount int) (status int) {
	reqBody := api.RedirectRequestBody{
		Amount: int32(amount),
		Target: target,
	}

	path := api.GetRedirectPath(deploymentID)
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) RemoveRedirect(deploymentID string) (status int) {
	path := api.GetRedirectPath(deploymentID)
	req := internalUtils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) GetRedirected(deploymentID string) (redirected int32, status int) {
	path := api.GetRedirectedPath(deploymentID)
	req := internalUtils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, &redirected)

	return
}

func (c *Client) SetResolvingAnswer(id string, resolved *api.ResolvedDTO) (status int) {
	reqBody := api.SetResolutionAnswerRequestBody{
		Resolved: resolved,
		ID:       id,
	}

	path := api.SetResolvingAnswerPath
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) GetLoad(deploymentID string) (load int, status int) {
	path := api.GetLoadPath(deploymentID)
	req := internalUtils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, &load)

	return
}

func (c *Client) GetClientCentroids(deploymentID string) (centroids []s2.CellID, status int) {
	path := api.GetAvgClientLocationPath(deploymentID)
	req := internalUtils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, &centroids)

	return
}

func (c *Client) SetExploringCells(deploymentID string, cells []s2.CellID) (status int) {
	reqBody := cells

	path := api.GetSetExploringClientLocationPath(deploymentID)
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) AddDeploymentNode(
	deploymentID string, node *utils.Node, location s2.CellID,
	exploring bool,
) (status int) {
	reqBody := api.AddDeploymentNodeRequestBody{
		Node:      node,
		Location:  location,
		Exploring: exploring,
	}

	path := api.GetAddDeploymentNodePath(deploymentID)
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) DeleteDeploymentNode(deploymentID string, nodeID string) (status int) {
	path := api.GetRemoveDeploymentNodePath(deploymentID, nodeID)
	req := internalUtils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) CanRedirectToYou(deploymentID, nodeID string) (can bool, status int) {
	path := api.GetRedirectingToYouPath(deploymentID, nodeID)
	req := internalUtils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)
	can = status == http.StatusOK

	return
}

func (c *Client) WillRedirectToYou(deploymentID, nodeID string) (status int) {
	path := api.GetRedirectingToYouPath(deploymentID, nodeID)
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) StopRedirectingToYou(deploymentID, nodeID string) (status int) {
	path := api.GetRedirectingToYouPath(deploymentID, nodeID)
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

		var bodyBytes []byte

		bodyBytes, err = json.Marshal(reqBody)
		if err != nil {
			panic(err)
		}

		req.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		req.ContentLength = int64(len(bodyBytes))
	}

	c.SetHostPort(req.URL.Host)

	return nil
}
