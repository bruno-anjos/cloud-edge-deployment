package client

import (
	"net/http"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/scheduler"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/docker/go-connections/nat"
)

type Client struct {
	*utils.GenericClient
}

func NewSchedulerClient(addr string) *Client {
	return &Client{
		GenericClient: utils.NewGenericClient(addr),
	}
}

func (c *Client) StartInstance(deploymentName, imageName string, ports nat.PortSet, replicaNum int, static bool,
	envVars []string, command []string) (status int) {
	reqBody := api.StartInstanceRequestBody{
		DeploymentName: deploymentName,
		ImageName:      imageName,
		Ports:          ports,
		Static:         static,
		EnvVars:        envVars,
		Command:        command,
	}

	path := api.GetInstancesPath()
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) StopInstance(instanceId string) (status int) {
	path := api.GetInstancePath(instanceId)
	req := utils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) StopAllInstances() (status int) {
	path := api.GetInstancesPath()
	req := utils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}
