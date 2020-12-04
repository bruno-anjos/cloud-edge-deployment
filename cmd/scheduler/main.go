package main

import (
	"flag"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/scheduler"
	internal "github.com/bruno-anjos/cloud-edge-deployment/internal/scheduler"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer/client_factory"
)

const (
	serviceName = "SCHEDULER"
)

func main() {
	debug := flag.Bool("d", false, "add debug logs")
	listenAddr := flag.String("l", utils.LocalhostAddr, "address to listen on")
	flag.Parse()

	deplFactory := &client_factory.ClientFactory{}
	internal.InitServer(deplFactory)
	utils.StartServerWithoutDefaultFlags(serviceName, utils.SchedulerPort, api.PrefixPath, internal.Routes, debug,
		listenAddr)
}
