package main

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/scheduler"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

const (
	serviceName = "SCHEDULER"
)

func main() {
	utils.StartServer(serviceName, scheduler.DefaultHostPort, scheduler.Port, scheduler.PrefixPath,
		scheduler.Routes)
}
