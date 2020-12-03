package main

import (
	api "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

const (
	serviceName = "ARCHIMEDES"
)

func main() {
	utils.StartServer(serviceName, utils.ArchimedesPort, api.PrefixPath, archimedes.Routes)
}
