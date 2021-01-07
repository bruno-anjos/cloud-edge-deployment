package archimedes

import (
	"github.com/docker/go-connections/nat"
)

type Deployment struct {
	ID    string
	Ports nat.PortSet
}

type Instance struct {
	ID              string
	DeploymentID    string
	IP              string
	PortTranslation nat.PortMap
	Initialized     bool
	Static          bool
	Local           bool
	Hops            int
}
