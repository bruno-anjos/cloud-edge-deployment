package autonomic

import (
	publicUtils "github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
)

type (
	AddServiceRequestBody  = serviceConfig
	ClosestNodeRequestBody = struct {
		Location  *publicUtils.Location
		ToExclude map[string]interface{}
	}
)
