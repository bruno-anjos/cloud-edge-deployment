package scheduler

import (
	internalUtils "github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/docker/go-connections/nat"
)

const (
	Port = internalUtils.SchedulerPort
)

type Client interface {
	utils.GenericClient
	StartInstance(deploymentName, imageName string, ports nat.PortSet, replicaNum int, static bool,
		envVars []string, command []string) (status int)
	StopInstance(instanceId string) (status int)
	StopAllInstances() (status int)
}

type ClientFactory interface {
	New(addr string) Client
}
