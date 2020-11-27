package autonomic

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/golang/geo/s2"
)

type (
	AddDeploymentRequestBody = deploymentConfig
	ClosestNodeRequestBody   struct {
		Locations []s2.CellID
		ToExclude map[string]interface{}
	}
	BlacklistNodeRequestBody struct {
		Origin string
		Nodes  []string
	}
	AddDeploymentChildRequestBody  = utils.Node
	SetDeploymentParentRequestBody = utils.Node
)
