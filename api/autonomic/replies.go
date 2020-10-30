package autonomic

import (
	"github.com/golang/geo/s2"
)

type (
	GetAllDeploymentsResponseBody = map[string]*DeploymentDTO
	ClosestNodeResponseBody       = string
	GetVicinityResponseBody       = map[string]s2.CellID
	GetMyLocationResponseBody     = s2.CellID
)
