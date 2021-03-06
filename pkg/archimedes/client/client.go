package client

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"sync"

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
	addr     string
	addrLock *sync.RWMutex
}

func NewArchimedesClient(addr string) *Client {
	newClient := client.NewGenericClient()
	archClient := &Client{
		GenericClient: newClient,
		addr:          addr,
		addrLock:      &sync.RWMutex{},
	}

	newClient.Client.CheckRedirect = archClient.handleRedirect

	return archClient
}

func (c *Client) RegisterDeployment(addr, deploymentID string, ports nat.PortSet, host *utils.Node) (status int) {
	reqBody := api.RegisterDeploymentRequestBody{
		Deployment: &api.DeploymentDTO{Ports: ports},
		Host:       host,
	}

	path := api.GetDeploymentPath(deploymentID)
	req := internalUtils.BuildRequest(http.MethodPost, addr, path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) RegisterDeploymentInstance(addr,
	deploymentID, instanceID string, static bool, portTranslation nat.PortMap, local bool,
) (status int) {
	reqBody := api.RegisterDeploymentInstanceRequestBody{
		Static:          static,
		PortTranslation: portTranslation,
		Local:           local,
	}

	path := api.GetDeploymentInstancePath(deploymentID, instanceID)
	req := internalUtils.BuildRequest(http.MethodPost, addr, path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) DeleteDeployment(addr, deploymentID string) (status int) {
	path := api.GetDeploymentPath(deploymentID)
	req := internalUtils.BuildRequest(http.MethodDelete, addr, path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) DeleteDeploymentInstance(addr, deploymentID, instanceID string) (status int) {
	path := api.GetDeploymentInstancePath(deploymentID, instanceID)
	req := internalUtils.BuildRequest(http.MethodDelete, addr, path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) GetDeployments(addr string) (deployments map[string]*api.Deployment, status int) {
	req := internalUtils.BuildRequest(http.MethodGet, addr, api.GetDeploymentsPath(), nil)

	deployments = api.GetAllDeploymentsResponseBody{}
	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, &deployments)

	return
}

func (c *Client) GetDeployment(addr, deploymentID string) (instances map[string]*api.Instance, status int) {
	req := internalUtils.BuildRequest(http.MethodGet, addr, api.GetDeploymentPath(deploymentID), nil)

	instances = api.GetDeploymentResponseBody{}
	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, &instances)

	return
}

func (c *Client) GetDeploymentInstance(addr, deploymentID, instanceID string) (instance *api.Instance, status int) {
	path := api.GetDeploymentInstancePath(deploymentID, instanceID)
	req := internalUtils.BuildRequest(http.MethodGet, addr, path, nil)

	instance = &api.GetDeploymentInstanceResponseBody{}
	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, instance)

	return
}

func (c *Client) GetInstance(addr, instanceID string) (instance *api.Instance, status int) {
	path := api.GetInstancePath(instanceID)
	req := internalUtils.BuildRequest(http.MethodGet, addr, path, nil)

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
	c.addrLock.RLock()
	addr := c.addr
	c.addrLock.RUnlock()
	log.Infof("resolving %s:%s using archimedes at %s", host, port.Port(), addr)
	req := internalUtils.BuildRequest(http.MethodPost, addr, path, reqBody)

	var resp api.ResolveResponseBody
	status, timedOut = internalUtils.DoRequest(c.GetHTTPClient(), req, &resp)
	log.Warnf("timed out on request %s:%s at %s", host, port.Port(), addr)

	if !timedOut {
		rHost = resp.Host
		rPort = resp.Port
	}

	return
}

func (c *Client) Redirect(addr, deploymentID, target string, amount int) (status int) {
	reqBody := api.RedirectRequestBody{
		Amount: int32(amount),
		Target: target,
	}

	path := api.GetRedirectPath(deploymentID)
	req := internalUtils.BuildRequest(http.MethodPost, addr, path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) RemoveRedirect(addr, deploymentID string) (status int) {
	path := api.GetRedirectPath(deploymentID)
	req := internalUtils.BuildRequest(http.MethodDelete, addr, path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) GetRedirected(addr, deploymentID string) (redirected int32, status int) {
	path := api.GetRedirectedPath(deploymentID)
	req := internalUtils.BuildRequest(http.MethodGet, addr, path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, &redirected)

	return
}

func (c *Client) SetResolvingAnswer(addr, id string, resolved *api.ResolvedDTO) (status int) {
	reqBody := api.SetResolutionAnswerRequestBody{
		Resolved: resolved,
		ID:       id,
	}

	path := api.SetResolvingAnswerPath
	req := internalUtils.BuildRequest(http.MethodPost, addr, path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) SetExploringCells(addr, deploymentID string, cells []s2.CellID) (status int) {
	reqBody := cells

	path := api.GetSetExploringClientLocationPath(deploymentID)
	req := internalUtils.BuildRequest(http.MethodPost, addr, path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) AddDeploymentNode(addr, deploymentID string, node *utils.Node, location s2.CellID, exploring bool,
) (status int) {
	reqBody := api.AddDeploymentNodeRequestBody{
		Node:      node,
		Location:  location,
		Exploring: exploring,
	}

	path := api.GetAddDeploymentNodePath(deploymentID)
	req := internalUtils.BuildRequest(http.MethodPost, addr, path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) DeleteDeploymentNode(addr, deploymentID, nodeID string) (status int) {
	path := api.GetRemoveDeploymentNodePath(deploymentID, nodeID)
	req := internalUtils.BuildRequest(http.MethodDelete, addr, path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) CanRedirectToYou(addr, deploymentID, nodeID string) (can bool, status int) {
	path := api.GetRedirectingToYouPath(deploymentID, nodeID)
	req := internalUtils.BuildRequest(http.MethodGet, addr, path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)
	can = status == http.StatusOK

	return
}

func (c *Client) WillRedirectToYou(addr, deploymentID, nodeID string) (status int) {
	path := api.GetRedirectingToYouPath(deploymentID, nodeID)
	req := internalUtils.BuildRequest(http.MethodPost, addr, path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) StopRedirectingToYou(addr, deploymentID, nodeID string) (status int) {
	path := api.GetRedirectingToYouPath(deploymentID, nodeID)
	req := internalUtils.BuildRequest(http.MethodDelete, addr, path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) ChangeArchimedesAddr(addr string) {
	c.addrLock.Lock()
	c.addr = addr
	c.addrLock.Unlock()
}

func (c *Client) handleRedirect(req *http.Request, via []*http.Request) error {
	log.Infof("redirecting %s to %s; redirections:", via[len(via)-1].URL.Host, req.URL.Host)

	if req.URL.Path == "/archimedes/resolve" {
		reqBody := api.ResolveRequestBody{}

		err := json.NewDecoder(req.Body).Decode(&reqBody)
		if err != nil {
			panic(err)
		}

		reqBody.Redirects = make([]string, len(via))
		for i, viaReq := range via {
			host, _, splitErr := net.SplitHostPort(viaReq.URL.Host)
			if splitErr != nil {
				log.Panic(splitErr)
			}
			reqBody.Redirects[i] = host
		}

		log.Infof("redirections %+v", reqBody.Redirects)

		var bodyBytes []byte

		bodyBytes, err = json.Marshal(reqBody)
		if err != nil {
			panic(err)
		}

		req.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		req.ContentLength = int64(len(bodyBytes))
	}

	c.addrLock.Lock()
	c.addr = req.URL.Host
	c.addrLock.Unlock()

	return nil
}
