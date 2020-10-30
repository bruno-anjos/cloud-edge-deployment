package autonomic

import (
	"github.com/golang/geo/s2"
)

type (
	AddDeploymentRequestBody = deploymentConfig
	ClosestNodeRequestBody   = struct {
		Location  s2.CellID
		ToExclude map[string]interface{}
	}
)
