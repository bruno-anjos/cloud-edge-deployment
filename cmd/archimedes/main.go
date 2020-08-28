package main

import (
	internal "github.com/bruno-anjos/cloud-edge-deployment/internal/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
)

const (
	serviceName = "ARCHIMEDES"
)

func main() {
	utils.StartServer(serviceName, archimedes.DefaultHostPort, archimedes.Port, internal.PrefixPath,
		internal.Routes)
}
