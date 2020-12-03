package scheduler

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/docker/go-connections/nat"
)

const (
	SchedulerPort = utils.SchedulerPort
)

type Client interface {
	StartInstance(deploymentName, imageName string, ports nat.PortSet, replicaNum int, static bool,
		envVars []string, command []string) (status int)
	StopInstance(instanceId string) (status int)
	StopAllInstances() (status int)
}
type ClientFactory interface {
	New(addr string) Client
}
