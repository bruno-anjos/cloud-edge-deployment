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

func NewSchedulerClient() *Client {
	return &Client{
		GenericClient: client.NewGenericClient(),
	}
}

func (c *Client) StartInstance(addr, deploymentName, imageName, instanceName string, ports nat.PortSet,
	replicaNum int, static bool, envVars, command []string) (status int) {
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
	req := internalUtils.BuildRequest(http.MethodPost, addr, path, reqBody)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) StopInstance(addr, instanceID, ip, removePath string) (status int) {
	path := api.GetInstancePath(instanceID)
	body := api.StopInstanceRequestBody{
		RemovePath: removePath,
		URL:        ip,
	}

	req := internalUtils.BuildRequest(http.MethodDelete, addr, path, body)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}

func (c *Client) StopAllInstances(addr string) (status int) {
	path := api.GetInstancesPath()
	req := internalUtils.BuildRequest(http.MethodDelete, addr, path, nil)

	status, _ = internalUtils.DoRequest(c.GetHTTPClient(), req, nil)

	return
}
