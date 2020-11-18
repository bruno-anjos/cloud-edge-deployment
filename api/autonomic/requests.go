package autonomic

import (
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
)
