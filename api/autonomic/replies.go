package autonomic

import (
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/golang/geo/s2"
)

type (
	Vicinity struct {
		Nodes     map[string]*utils.Node
		Locations map[string]s2.CellID
	}

	GetAllDeploymentsResponseBody = map[string]*DeploymentDTO
	ClosestNodeResponseBody       = utils.Node
	GetVicinityResponseBody       = Vicinity
	GetMyLocationResponseBody     = s2.CellID
)
