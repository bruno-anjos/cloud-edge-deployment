package scheduler

import (
	"net/http"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/scheduler"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/scheduler"
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

func (c *Client) StartInstance(serviceName, imageName string, ports nat.PortSet, static bool,
	envVars []string) (status int) {
	reqBody := api.StartInstanceRequestBody{
		ServiceName: serviceName,
		ImageName:   imageName,
		Ports:       ports,
		Static:      static,
		EnvVars:     envVars,
	}

	path := scheduler.GetInstancesPath()
	req := utils.BuildRequest(http.MethodPost, c.GetHostPort(), path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) StopInstance(instanceId string) (status int) {
	path := scheduler.GetInstancePath(instanceId)
	req := utils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *Client) StopAllInstances() (status int) {
	path := scheduler.GetInstancesPath()
	req := utils.BuildRequest(http.MethodDelete, c.GetHostPort(), path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}
