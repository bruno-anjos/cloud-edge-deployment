package main

import (
	scheduler2 "github.com/bruno-anjos/cloud-edge-deployment/api/scheduler"
	internal "github.com/bruno-anjos/cloud-edge-deployment/internal/scheduler"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/scheduler"
)

const (
	serviceName = "SCHEDULER"
)

func main() {
	utils.StartServer(serviceName, scheduler.DefaultHostPort, scheduler.Port, scheduler2.PrefixPath,
		internal.Routes)
}
