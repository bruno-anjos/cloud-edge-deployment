package deployer

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

type HierarchyEntryDTO struct {
	Parent      *utils.Node
	Grandparent *utils.Node
	Children    map[string]utils.Node
	Static      bool
	IsOrphan    bool
}

type (
	DeploymentDTO struct {
		Children            []*utils.Node
		Parent              *utils.Node
		Grandparent         *utils.Node
		DeploymentId        string
		Static              bool
		DeploymentYAMLBytes []byte
	}

	DeploymentYAML struct {
		Spec struct {
			Replicas       int
			DeploymentName string `yaml:"serviceName"`
			Template       struct {
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

	MigrateDTO struct {
		Origin string
		Target string
	}
)
