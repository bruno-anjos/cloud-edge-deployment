package autonomic

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/golang/geo/s2"
)

type (
	GetAllDeploymentsResponseBody = map[string]*DeploymentDTO
	ClosestNodeResponseBody       = utils.Node
	GetVicinityResponseBody       = autonomic.Vicinity
	GetMyLocationResponseBody     = s2.CellID
)
