package scheduler

import (
	"github.com/docker/go-connections/nat"
)

type ContainerInstanceDTO struct {
	DeploymentName string `json:"service_name"`
	ImageName      string `json:"image_name"`
	Command        []string
	Ports          nat.PortSet
	Static         bool
	EnvVars        []string
}
