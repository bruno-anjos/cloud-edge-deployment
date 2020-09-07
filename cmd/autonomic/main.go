package main

import (
	autonomic2 "github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
	internal "github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
)

const (
	serviceName = "AUTONOMIC"
)

func main() {
	utils.StartServer(serviceName, autonomic.DefaultHostPort, autonomic.Port, autonomic2.PrefixPath, internal.Routes)
}
