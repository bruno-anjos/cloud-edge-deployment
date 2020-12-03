package main

import (
	deployer2 "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	internal "github.com/bruno-anjos/cloud-edge-deployment/internal/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

const (
	serviceName = "DEPLOYER"
)

func main() {
	utils.StartServer(serviceName, utils.DeployerPort, deployer2.PrefixPath, internal.Routes)
}
