package client

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils/client"
	"github.com/golang/geo/s2"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	internalUtils "github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	HeartbeatCheckerTimeout = 60
)

type Client struct {
	utils.GenericClient
}

func NewDeployerClient() *Client {
	return &Client{
		GenericClient: client.NewGenericClient(),
	}
}

func (c *Client) GetDeployments(addr string) (deploymentIds []string, status int) {
	path := api.GetDeploymentsPath()
	req := internalUtils.BuildRequest(http.MethodGet, addr, path, nil)

	var resp api.GetDeploymentsResponseBody
	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, &resp)
	deploymentIds = resp

	return
}

func (c *Client) RegisterDeployment(addr, deploymentID string, static bool, deploymentYamlBytes []byte,
	grandparent, parent *utils.Node, children []*utils.Node, exploringTTL int) (status int) {
	reqBody := api.RegisterDeploymentRequestBody{
		DeploymentConfig: &api.DeploymentDTO{
			Children:            children,
			Parent:              parent,
			Grandparent:         grandparent,
			DeploymentID:        deploymentID,
			Static:              static,
			DeploymentYAMLBytes: deploymentYamlBytes,
		},
		ExploringTTL: exploringTTL,
	}
	path := api.GetDeploymentsPath()
	req := internalUtils.BuildRequest(http.MethodPost, addr, path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) ExtendDeploymentTo(addr, deploymentID string, node *utils.Node, exploringTTL int,
	config *api.ExtendDeploymentConfig) (status int) {
	reqBody := api.ExtendDeploymentRequestBody{
		Node:         node,
		ExploringTTL: exploringTTL,
		Config:       config,
	}

	path := api.GetExtendDeploymentPath(deploymentID)
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

func (c *Client) RegisterDeploymentInstance(addr, deploymentID, instanceID string, static bool,
	portTranslation nat.PortMap, local bool) (status int) {
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

func (c *Client) RegisterHearbeatDeploymentInstance(addr, deploymentID, instanceID string) (status int) {
	path := api.GetDeploymentInstanceAlivePath(deploymentID, instanceID)
	req := internalUtils.BuildRequest(http.MethodPost, addr, path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) SendHeartbeatDeploymentInstance(addr, deploymentID, instanceID string) (status int) {
	path := api.GetDeploymentInstanceAlivePath(deploymentID, instanceID)
	req := internalUtils.BuildRequest(http.MethodPut, addr, path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) WarnOfDeadChild(addr, deploymentID, deadChildID string, grandChild *utils.Node,
	alternatives map[string]*utils.Node, locations []s2.CellID) (status int) {
	var reqBody api.DeadChildRequestBody
	reqBody.Grandchild = grandChild
	reqBody.Alternatives = alternatives
	reqBody.Locations = locations

	path := api.GetDeadChildPath(deploymentID, deadChildID)
	req := internalUtils.BuildRequest(http.MethodPost, addr, path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) SetGrandparent(addr, deploymentID string, grandparent *utils.Node) (status int) {
	reqBody := *grandparent

	path := api.GetSetGrandparentPath(deploymentID)
	req := internalUtils.BuildRequest(http.MethodPost, addr, path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) WarnThatIAmParent(addr, deploymentID string, parent, grandparent *utils.Node) (status int) {
	reqBody := api.IAmYourParentRequestBody{
		Parent:      parent,
		Grandparent: grandparent,
	}

	path := api.GetImYourParentPath(deploymentID)
	req := internalUtils.BuildRequest(http.MethodPost, addr, path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) ChildDeletedDeployment(addr, deploymentID, childID string) (status int) {
	path := api.GetDeploymentChildPath(deploymentID, childID)
	req := internalUtils.BuildRequest(http.MethodDelete, addr, path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) GetHierarchyTable(addr string) (table map[string]*api.HierarchyEntryDTO, status int) {
	path := api.GetHierarchyTablePath()
	req := internalUtils.BuildRequest(http.MethodGet, addr, path, nil)

	var resp api.GetHierarchyTableResponseBody
	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, &resp)

	table = resp

	return
}

func (c *Client) SetParentAlive(addr, parentID string) (status int) {
	path := api.GetParentAlivePath(parentID)
	req := internalUtils.BuildRequest(http.MethodPost, addr, path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) SendInstanceHeartbeatToDeployerPeriodically(addr string) {
	deploymentID := os.Getenv(utils.DeploymentEnvVarName)
	instanceID := os.Getenv(utils.InstanceEnvVarName)

	status := c.RegisterHearbeatDeploymentInstance(addr, deploymentID, instanceID)
	switch status {
	case http.StatusConflict:
		log.Infof("deployment %s instance %s already has a heartbeat sender", deploymentID, instanceID)

		return
	case http.StatusOK:
	default:
		panic(errors.New(fmt.Sprintf("received unexpected status %d", status)))
	}

	ticker := time.NewTicker((HeartbeatCheckerTimeout / 3) * time.Second)

	for {
		<-ticker.C
		log.Info("sending heartbeat to deployer")

		status = c.SendHeartbeatDeploymentInstance(addr, deploymentID, instanceID)
		switch status {
		case http.StatusNotFound:
			log.Warnf("heartbeat to deployer retrieved not found")
		case http.StatusOK:
		default:
			log.Error(errors.New(fmt.Sprintf("received unexpected status %d", status)))
		}
	}
}

func (c *Client) SendAlternatives(addr, myID string, alternatives []*utils.Node) (status int) {
	reqBody := alternatives

	path := api.GetSetAlternativesPath(myID)
	req := internalUtils.BuildRequest(http.MethodPost, addr, path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) Fallback(addr, deploymentID string, orphan *utils.Node, orphanLocation s2.CellID) (status int) {
	var reqBody api.FallbackRequestBody
	reqBody.Orphan = orphan
	reqBody.OrphanLocation = orphanLocation

	path := api.GetFallbackPath(deploymentID)
	req := internalUtils.BuildRequest(http.MethodPost, addr, path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) GetFallback(addr string) (fallback *utils.Node, status int) {
	path := api.GetGetFallbackIDPath()
	req := internalUtils.BuildRequest(http.MethodGet, addr, path, nil)

	var respBody api.GetFallbackResponseBody
	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, &respBody)

	fallback = &respBody

	return
}

func (c *Client) HasDeployment(addr, deploymentID string) (has bool, status int) {
	path := api.GetHasDeploymentPath(deploymentID)
	req := internalUtils.BuildRequest(http.MethodGet, addr, path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	has = status == http.StatusOK

	return
}

func (c *Client) PropagateLocationToHorizon(addr, deploymentID string, origin *utils.Node, location s2.CellID, tTL int8,
	op api.PropagateOpType) (status int) {
	reqBody := api.PropagateLocationToHorizonRequestBody{
		Operation: op,
		TTL:       tTL,
		Child:     origin,
		Location:  location,
	}

	path := api.GetPropagateLocationToHorizonPath(deploymentID)
	req := internalUtils.BuildRequest(http.MethodPost, addr, path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) SetReady(addr string) (status int) {
	path := api.GetSetReadyPath()
	req := internalUtils.BuildRequest(http.MethodPost, addr, path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}
