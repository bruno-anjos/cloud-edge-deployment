package main

import (
	internal "github.com/bruno-anjos/cloud-edge-deployment/internal/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
)

const (
	serviceName = "DEPLOYER"
)

func main() {
	utils.StartServer(serviceName, deployer.DefaultHostPort, deployer.Port, internal.PrefixPath, internal.Routes)
}
