package deployer

import (
	archimedes2 "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

const (
	ParentIdx      = 0
	GrandparentIdx = 1
)

type (
	ExpandTreeRequestBody      = string
	RegisterServiceRequestBody = DeploymentDTO
	AddNodeRequestBody         = string
	DeadChildRequestBody       = struct {
		Grandchild   *utils.Node
		Alternatives map[string]*utils.Node
	}
	TakeChildRequestBody               = utils.Node
	IAmYourParentRequestBody           = []*utils.Node
	RegisterServiceInstanceRequestBody = archimedes2.InstanceDTO
	AlternativesRequestBody            = []*utils.Node
)
