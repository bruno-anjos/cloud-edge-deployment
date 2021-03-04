package scheduler

import (
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/docker/go-connections/nat"
)

const (
	Port = 1501
)

type Client interface {
	utils.GenericClient
	StartInstance(addr, deploymentName, imageName, instanceName string, ports nat.PortSet, replicaNum int, static bool,
		envVars, command []string) (status int)
	StopInstance(addr, instanceID, ip, path string) (status int)
	StopAllInstances(addr string) (status int)
}

type ClientFactory interface {
	New() Client
}
