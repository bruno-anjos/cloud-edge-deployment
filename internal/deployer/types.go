package deployer

import (
	"sync"

	"github.com/docker/go-connections/nat"
)

type deployment struct {
	DeploymentID      string
	NumberOfInstances int
	Command           []string
	Image             string
	EnvVars           []string
	Ports             nat.PortSet
	Static            bool
	Lock              *sync.RWMutex
}

type pairDeploymentIDStatus struct {
	DeploymentID string
	IsUp         bool
	Mutex        *sync.Mutex
}
