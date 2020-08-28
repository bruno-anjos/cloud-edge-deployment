package main

import (
	internal "github.com/bruno-anjos/cloud-edge-deployment/internal/scheduler"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/scheduler"
)

const (
	serviceName = "SCHEDULER"
)

func main() {
	utils.StartServer(serviceName, scheduler.DefaultHostPort, scheduler.Port, internal.PrefixPath,
		internal.Routes)
}
