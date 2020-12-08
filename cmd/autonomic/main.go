package main

import (
	autonomicAPI "github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
	internal "github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/servers"
	archimedesFactory "github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes/client_factory"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	autonomicFactory "github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic/client_factory"
	deployerFactory "github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer/client_factory"
	schedulerFactory "github.com/bruno-anjos/cloud-edge-deployment/pkg/scheduler/client_factory"
)

const (
	serviceName = "AUTONOMIC"
)

func main() {
	autoFactory := &autonomicFactory.ClientFactory{}
	archFactory := &archimedesFactory.ClientFactory{}
	deplFactory := &deployerFactory.ClientFactory{}
	schedFactory := &schedulerFactory.ClientFactory{}

	internal.InitServer(autoFactory, archFactory, deplFactory, schedFactory)
	servers.StartServer(serviceName, autonomic.Port, autonomicAPI.PrefixPath, internal.Routes)
}
