package archimedes

import (
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
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
	Host         *utils.Node
	Deployment   *Deployment
	Instances    map[string]*Instance
	NumberOfHops int
	MaxHops      int
	Version      int
}

type DiscoverMsg struct {
	MessageID    uuid.UUID
	Origin       string
	NeighborSent string
	Entries      map[string]*DeploymentsTableEntryDTO
}

type ToResolveDTO struct {
	Host string
	Port nat.Port
}

type redirectDTO struct {
	Amount int32
	Target string
}
