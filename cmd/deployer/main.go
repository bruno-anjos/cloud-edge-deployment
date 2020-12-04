package main

import (
	deployerAPI "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	internal "github.com/bruno-anjos/cloud-edge-deployment/internal/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	archimedesFactory "github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes/client_factory"
	autonomicFactory "github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic/client_factory"
	deployerFactory "github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer/client_factory"
	schedulerFactory "github.com/bruno-anjos/cloud-edge-deployment/pkg/scheduler/client_factory"
)

const (
	serviceName = "DEPLOYER"
)

func main() {
	autoFactory := &autonomicFactory.ClientFactory{}
	archFactory := &archimedesFactory.ClientFactory{}
	deplFactory := &deployerFactory.ClientFactory{}
	schedFactory := &schedulerFactory.ClientFactory{}

	internal.InitServer(autoFactory, archFactory, deplFactory, schedFactory)
	utils.StartServer(serviceName, utils.DeployerPort, deployerAPI.PrefixPath, internal.Routes)
}
