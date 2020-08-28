package main

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

const (
	serviceName = "ARCHIMEDES"
)

func main() {
	utils.StartServer(serviceName, archimedes.DefaultHostPort, archimedes.Port, archimedes.PrefixPath,
		archimedes.Routes)
}
