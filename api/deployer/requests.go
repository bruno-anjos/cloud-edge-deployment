package deployer

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
)

type (
	ExpandTreeRequestBody              = string
	RegisterServiceRequestBody         = deployer.DeploymentDTO
	AddNodeRequestBody                 = string
	DeadChildRequestBody               = utils.Node
	TakeChildRequestBody               = utils.Node
	IAmYourParentRequestBody           = utils.Node
	RegisterServiceInstanceRequestBody = archimedes.InstanceDTO
	MigrateDeploymentBody              = deployer.MigrateDTO
)
