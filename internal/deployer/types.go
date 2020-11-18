package deployer

import (
	"sync"

	"github.com/docker/go-connections/nat"
)

type Deployment struct {
	DeploymentId      string
	NumberOfInstances int
	Command           []string
	Image             string
	EnvVars           []string
	Ports             nat.PortSet
	Static            bool
	Lock              *sync.RWMutex
}

type PairDeploymentIdStatus struct {
	DeploymentId string
	IsUp         bool
	Mutex        *sync.Mutex
}
