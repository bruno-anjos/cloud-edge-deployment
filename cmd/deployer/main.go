package main

import (
	deployer2 "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	internal "github.com/bruno-anjos/cloud-edge-deployment/internal/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
)

const (
	serviceName = "DEPLOYER"
)

func main() {
	utils.StartServer(serviceName, deployer.DefaultHostPort, deployer.Port, deployer2.PrefixPath, internal.Routes)
}
