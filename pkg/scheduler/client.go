package scheduler

import (
	"net/http"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/scheduler"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/scheduler"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/docker/go-connections/nat"
)

type SchedulerClient struct {
	*utils.GenericClient
}

func NewSchedulerClient(addr string) *SchedulerClient {
	return &SchedulerClient{
		GenericClient: utils.NewGenericClient(addr, scheduler.Port),
	}
}

func (c *SchedulerClient) StartInstance(serviceName, imageName string, ports nat.PortSet, static bool,
	envVars []string) (status int) {
	reqBody := api.StartInstanceRequestBody{
		ServiceName: serviceName,
		ImageName:   imageName,
		Ports:       ports,
		Static:      static,
		EnvVars:     envVars,
	}

	path := scheduler.GetInstancesPath()
	req := utils.BuildRequest(http.MethodPost, c.HostPort, path, reqBody)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *SchedulerClient) StopInstance(instanceId string) (status int) {
	path := scheduler.GetInstancePath(instanceId)
	req := utils.BuildRequest(http.MethodDelete, c.HostPort, path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}

func (c *SchedulerClient) StopAllInstances() (status int) {
	path := scheduler.GetInstancesPath()
	req := utils.BuildRequest(http.MethodDelete, c.HostPort, path, nil)

	status, _ = utils.DoRequest(c.Client, req, nil)

	return
}
