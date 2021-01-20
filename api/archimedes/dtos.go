package archimedes

import (
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/docker/go-connections/nat"
)

type ResolvedDTO struct {
	Host string
	Port string
}

type DeploymentDTO struct {
	Ports nat.PortSet
}

type InstanceDTO struct {
	PortTranslation nat.PortMap `json:"port_translation"`
	Static          bool
	Local           bool
}

type DeploymentsTableEntryDTO struct {
	Deployment *Deployment
	Instances  map[string]*Instance
	MaxHops    int
}

type DiscoverMsg struct {
	MessageID string
	Origin    *utils.Node
	Entries   map[string]*DeploymentsTableEntryDTO
}

const (
	DiscoverMessageID = iota
)

type ToResolveDTO struct {
	Host string
	Port nat.Port
}

type redirectDTO struct {
	Amount int32
	Target string
}
