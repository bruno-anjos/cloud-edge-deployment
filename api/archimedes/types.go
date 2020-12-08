package archimedes

import (
	"github.com/docker/go-connections/nat"
)

type Deployment struct {
	Id    string
	Ports nat.PortSet
}

type Instance struct {
	Id              string
	DeploymentId    string
	Ip              string
	PortTranslation nat.PortMap
	Initialized     bool
	Static          bool
	Local           bool
}
