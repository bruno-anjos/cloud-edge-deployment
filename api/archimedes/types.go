package archimedes

import (
	"github.com/docker/go-connections/nat"
)

type Service struct {
	Id    string
	Ports nat.PortSet
}

func (s *Service) ToTransfarable() *Service {
	return &Service{
		Id:    s.Id,
		Ports: s.Ports,
	}
}

type Instance struct {
	Id              string
	ServiceId       string
	Ip              string
	PortTranslation nat.PortMap
	Initialized     bool
	Static          bool
	Local           bool
}
