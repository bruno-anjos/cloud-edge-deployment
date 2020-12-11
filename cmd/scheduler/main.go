package main

import (
	"flag"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/scheduler"
	internal "github.com/bruno-anjos/cloud-edge-deployment/internal/scheduler"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/servers"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer/clientfactory"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/scheduler"
)

const (
	serviceName = "SCHEDULER"
)

func main() {
	debug := flag.Bool("d", false, "add debug logs")
	listenAddr := flag.String("l", servers.LocalhostAddr, "address to listen on")
	flag.Parse()

	deplFactory := &clientfactory.ClientFactory{}
	internal.InitServer(deplFactory)
	servers.StartServerWithoutDefaultFlags(serviceName, scheduler.Port, api.PrefixPath, internal.Routes, debug,
		listenAddr)
}
