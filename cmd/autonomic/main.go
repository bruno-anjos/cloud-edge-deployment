package main

import (
	autonomicAPI "github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
	internal "github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	archimedesFactory "github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes/client_factory"
	autonomicFactory "github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic/client_factory"
	deployerFactory "github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer/client_factory"
)

const (
	serviceName = "AUTONOMIC"
)

func main() {
	autoFactory := &autonomicFactory.ClientFactory{}
	archFactory := &archimedesFactory.ClientFactory{}
	deplFactory := &deployerFactory.ClientFactory{}

	internal.InitServer(autoFactory, archFactory, deplFactory)
	utils.StartServer(serviceName, utils.AutonomicPort, autonomicAPI.PrefixPath, internal.Routes)
}
