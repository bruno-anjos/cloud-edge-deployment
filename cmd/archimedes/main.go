package main

import (
	archimedes2 "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	internal "github.com/bruno-anjos/cloud-edge-deployment/internal/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
)

const (
	serviceName = "ARCHIMEDES"
)

func main() {
	utils.StartServer(serviceName, archimedes.DefaultHostPort, archimedes.Port, archimedes2.PrefixPath,
		internal.Routes)
}
