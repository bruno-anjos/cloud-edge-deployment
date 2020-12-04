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

func NewDeployerClient(addr string) *Client {
	return &Client{
		GenericClient: client.NewGenericClient(addr),
	}
}

func (c *Client) GetDeployments() (deploymentIds []string, status int) {
	path := api.GetDeploymentsPath()
	req := internalUtils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	var resp api.GetDeploymentsResponseBody
	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, &resp)
	deploymentIds = resp

	return
}

func (c *Client) RegisterDeployment(deploymentId string, static bool, deploymentYamlBytes []byte,
	grandparent, parent *internalUtils.Node, children []*internalUtils.Node, exploringTTL int) (status int) {
	reqBody := api.RegisterDeploymentRequestBody{
		DeploymentConfig: &api.DeploymentDTO{
			Children:            children,
			Parent:              parent,
			Grandparent:         grandparent,
			DeploymentId:        deploymentId,
			Static:              static,
			DeploymentYAMLBytes: deploymentYamlBytes,
		},
		ExploringTTL: exploringTTL,
	}
	path := api.GetDeploymentsPath()
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) ExtendDeploymentTo(deploymentId string, node *internalUtils.Node, exploringTTL int,
	config *api.ExtendDeploymentConfig) (status int) {
	reqBody := api.ExtendDeploymentRequestBody{
		Node:         node,
		ExploringTTL: exploringTTL,
		Config:       config,
	}

	path := api.GetExtendDeploymentPath(deploymentId)
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

func (c *Client) RegisterHearbeatDeploymentInstance(deploymentId, instanceId string) (status int) {
	path := api.GetDeploymentInstanceAlivePath(deploymentId, instanceId)
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) SendHearbeatDeploymentInstance(deploymentId, instanceId string) (status int) {
	path := api.GetDeploymentInstanceAlivePath(deploymentId, instanceId)
	req := internalUtils.BuildRequest(http.MethodPut, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) WarnOfDeadChild(deploymentId, deadChildId string, grandChild *internalUtils.Node,
	alternatives map[string]*internalUtils.Node, locations []s2.CellID) (status int) {
	var reqBody api.DeadChildRequestBody
	reqBody.Grandchild = grandChild
	reqBody.Alternatives = alternatives
	reqBody.Locations = locations

	path := api.GetDeadChildPath(deploymentId, deadChildId)
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) SetGrandparent(deploymentId string, grandparent *internalUtils.Node) (status int) {
	var reqBody api.SetGrandparentRequestBody
	reqBody = *grandparent

	path := api.GetSetGrandparentPath(deploymentId)
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) WarnThatIAmParent(deploymentId string, parent, grandparent *internalUtils.Node) (status int) {
	reqBody := api.IAmYourParentRequestBody{
		Parent:      parent,
		Grandparent: grandparent,
	}

	path := api.GetImYourParentPath(deploymentId)
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) ChildDeletedDeployment(deploymentId, childId string) (status int) {
	path := api.GetDeploymentChildPath(deploymentId, childId)
	req := internalUtils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) GetHierarchyTable() (table map[string]*api.HierarchyEntryDTO, status int) {
	path := api.GetHierarchyTablePath()
	req := internalUtils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	var resp api.GetHierarchyTableResponseBody
	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, &resp)

	table = resp

	return
}

func (c *Client) SetParentAlive(parentId string) (status int) {
	path := api.GetParentAlivePath(parentId)
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) SendInstanceHeartbeatToDeployerPeriodically() {
	deploymentId := os.Getenv(utils.DeploymentEnvVarName)
	instanceId := os.Getenv(utils.InstanceEnvVarName)

	status := c.RegisterHearbeatDeploymentInstance(deploymentId, instanceId)
	switch status {
	case http.StatusConflict:
		log.Debugf("deployment %s instance %s already has a heartbeat sender", deploymentId, instanceId)
		return
	case http.StatusOK:
	default:
		panic(errors.New(fmt.Sprintf("received unexpected status %d", status)))
	}

	ticker := time.NewTicker((HeartbeatCheckerTimeout / 3) * time.Second)
	for {
		<-ticker.C
		log.Info("sending heartbeat to deployer")
		status = c.SendHearbeatDeploymentInstance(deploymentId, instanceId)
		switch status {
		case http.StatusNotFound:
			log.Warnf("heartbeat to deployer retrieved not found")
		case http.StatusOK:
		default:
			panic(errors.New(fmt.Sprintf("received unexpected status %d", status)))
		}
	}
}

func (c *Client) SendAlternatives(myId string, alternatives []*internalUtils.Node) (status int) {
	var reqBody api.AlternativesRequestBody
	reqBody = alternatives

	path := api.GetSetAlternativesPath(myId)
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) Fallback(deploymentId string, orphan *internalUtils.Node, orphanLocation s2.CellID) (status int) {
	var reqBody api.FallbackRequestBody
	reqBody.Orphan = orphan
	reqBody.OrphanLocation = orphanLocation

	path := api.GetFallbackPath(deploymentId)
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) GetFallback() (fallback *internalUtils.Node, status int) {
	path := api.GetGetFallbackIdPath()
	req := internalUtils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	var (
		respBody api.GetFallbackResponseBody
	)
	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, &respBody)

	fallback = &respBody

	return
}

func (c *Client) HasDeployment(deploymentId string) (has bool, status int) {
	path := api.GetHasDeploymentPath(deploymentId)
	req := internalUtils.BuildRequest(http.MethodGet, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	has = status == http.StatusOK
	return
}

func (c *Client) PropagateLocationToHorizon(deploymentId, origin string, location s2.CellID, TTL int8,
	op api.PropagateOpType) (status int) {
	reqBody := api.PropagateLocationToHorizonRequestBody{
		Operation: op,
		TTL:       TTL,
		ChildId:   origin,
		Location:  location,
	}

	path := api.GetPropagateLocationToHorizonPath(deploymentId)
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}
