package main

import (
	deployerAPI "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	internal "github.com/bruno-anjos/cloud-edge-deployment/internal/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/servers"
	archimedesFactory "github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes/clientfactory"
	autonomicFactory "github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic/clientfactory"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	deployerFactory "github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer/clientfactory"
	schedulerFactory "github.com/bruno-anjos/cloud-edge-deployment/pkg/scheduler/clientfactory"
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
	servers.StartServer(serviceName, deployer.Port, deployerAPI.PrefixPath, internal.Routes)
}
