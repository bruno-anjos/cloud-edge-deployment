package archimedes

import (
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/docker/go-connections/nat"
)

type Deployment struct {
	ID    string
	Ports nat.PortSet
}

type Instance struct {
	ID              string
	DeploymentID    string
	Host            *utils.Node
	PortTranslation nat.PortMap
	Initialized     bool
	Static          bool
	Local           bool
	Hops            int
}
