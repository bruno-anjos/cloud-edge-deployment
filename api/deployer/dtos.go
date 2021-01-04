package deployer

import (
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
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
		DeploymentID        string
		Static              bool
		DeploymentYAMLBytes []byte
	}

	DeploymentYAML struct {
		Replicas       int
		DeploymentName string `yaml:"deploymentName"`
		Containers     []struct {
			Image   string
			Command []string
			Ports   []struct {
				ContainerPort string `yaml:"containerPort"`
			}
			Env []struct {
				Name  string
				Value string
			}
		}
		Static          bool
		DepthFactor     float64  `yaml:"depthFactor"`
		InstanceNameFmt []string `yaml:"instanceNameFmt"`
		RemovePath      string   `yaml:"removePath"`
	}
)
