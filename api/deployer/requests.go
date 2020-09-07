package deployer

import (
	archimedes2 "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

type (
	ExpandTreeRequestBody              = string
	RegisterServiceRequestBody         = DeploymentDTO
	AddNodeRequestBody                 = string
	DeadChildRequestBody               = utils.Node
	TakeChildRequestBody               = utils.Node
	IAmYourParentRequestBody           = utils.Node
	RegisterServiceInstanceRequestBody = archimedes2.InstanceDTO
	MigrateDeploymentBody              = MigrateDTO
)
