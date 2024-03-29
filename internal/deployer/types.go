package deployer

import (
	"sync"

	"github.com/docker/go-connections/nat"
)

type Deployment struct {
	DeploymentId      string
	NumberOfInstances int
	Image             string
	EnvVars           []string
	Ports             nat.PortSet
	Static            bool
	Lock              *sync.RWMutex
}

type PairServiceIdStatus struct {
	ServiceId string
	IsUp      bool
	Mutex     *sync.Mutex
}
