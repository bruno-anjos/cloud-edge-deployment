package deployer

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

type HierarchyEntryDTO struct {
	Parent      *utils.Node
	Grandparent *utils.Node
	Child       map[string]*utils.Node
	Static      bool
	IsOrphan    bool
}

type DeploymentDTO struct {
	Parent              *utils.Node
	Grandparent         *utils.Node
	DeploymentId        string
	Static              bool
	DeploymentYAMLBytes []byte
}

type DeploymentYAML struct {
	Spec struct {
		Replicas    int
		ServiceName string `yaml:"serviceName"`
		Template    struct {
			Spec struct {
				Containers []struct {
					Image string
					Env   []struct {
						Name  string
						Value string
					}
					Ports []struct {
						ContainerPort string `yaml:"containerPort"`
					}
				}
			}
		}
	}
}
