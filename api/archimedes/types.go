package archimedes

import (
	"github.com/docker/go-connections/nat"
)

type Deployment struct {
	Id    string
	Ports nat.PortSet
}

func (s *Deployment) ToTransfarable() *Deployment {
	return &Deployment{
		Id:    s.Id,
		Ports: s.Ports,
	}
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
