package client

import (
	"net/http"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/scheduler"
	internalUtils "github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils/client"
	"github.com/docker/go-connections/nat"
)

type Client struct {
	utils.GenericClient
}

func NewSchedulerClient(addr string) *Client {
	return &Client{
		GenericClient: client.NewGenericClient(addr),
	}
}

func (c *Client) StartInstance(deploymentName, imageName, instanceName string, ports nat.PortSet,
	replicaNum int, static bool, envVars []string, command []string) (status int) {
	reqBody := api.StartInstanceRequestBody{
		DeploymentName: deploymentName,
		ImageName:      imageName,
		Command:        command,
		Ports:          ports,
		Static:         static,
		EnvVars:        envVars,
		ReplicaNumber:  replicaNum,
		InstanceName:   instanceName,
	}

	path := api.GetInstancesPath()
	req := internalUtils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) StopInstance(instanceID string) (status int) {
	path := api.GetInstancePath(instanceID)
	req := internalUtils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) StopAllInstances() (status int) {
	path := api.GetInstancesPath()
	req := internalUtils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}
