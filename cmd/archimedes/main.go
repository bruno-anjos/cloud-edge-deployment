package main

import (
	api "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	internal "github.com/bruno-anjos/cloud-edge-deployment/internal/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/servers"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	autonomicFactory "github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic/clientfactory"
	deployerFactory "github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer/clientfactory"
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
