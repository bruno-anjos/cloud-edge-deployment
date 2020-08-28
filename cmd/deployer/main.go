package main

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

const (
	serviceName = "DEPLOYER"
)

func main() {
	utils.StartServer(serviceName, deployer.DefaultHostPort, deployer.Port, deployer.PrefixPath, deployer.Routes)
}
