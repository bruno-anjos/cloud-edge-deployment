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
	Host       *utils.Node
	Deployment *Deployment
	Instances  map[string]*Instance
	MaxHops    int
	Version    int
}

type DiscoverMsg struct {
	MessageID uuid.UUID
	Origin    *utils.Node
	Entries   map[string]*DeploymentsTableEntryDTO
}

const (
	AddRemoteDeploymentMessageID = iota
	AddRemoteInstanceMessageID
	RemoveRemoteDeploymentMessageID
)

type AddRemoteDeploymentMsg struct {
	MessageID  uuid.UUID
	Origin     *utils.Node
	Deployment *Deployment
}

type AddRemoteInstanceMsg struct {
	MessageID uuid.UUID
	Origin    *utils.Node
	Instance  *Instance
}

type RemoveRemoteDeploymentMsg struct {
	MessageID    uuid.UUID
	Origin       *utils.Node
	DeploymentID string
}

type ToResolveDTO struct {
	Host string
	Port nat.Port
}

type redirectDTO struct {
	Amount int32
	Target string
}
