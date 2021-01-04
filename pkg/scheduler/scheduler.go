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
	StartInstance(deploymentName, imageName, instanceName string, ports nat.PortSet, replicaNum int, static bool,
		envVars, command []string) (status int)
	StopInstance(instanceID, ip, path string) (status int)
	StopAllInstances() (status int)
}

type ClientFactory interface {
	New(addr string) Client
}
