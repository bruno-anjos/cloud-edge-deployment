package main

import (
	api "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	internal "github.com/bruno-anjos/cloud-edge-deployment/internal/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/servers"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	autonomicFactory "github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic/client_factory"
	deployerFactory "github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer/client_factory"
)

const (
	serviceName = "ARCHIMEDES"
)

func main() {
	autoFactory := &autonomicFactory.ClientFactory{}
	deplFactory := &deployerFactory.ClientFactory{}

	internal.InitServer(autoFactory, deplFactory)
	servers.StartServer(serviceName, archimedes.Port, api.PrefixPath, internal.Routes)
}
