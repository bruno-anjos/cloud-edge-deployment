package scheduler

import (
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/docker/go-connections/nat"
)

const (
	Port = 50001
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
