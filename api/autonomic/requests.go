package autonomic

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

type (
	AddServiceRequestBody  = serviceConfig
	ClosestNodeRequestBody = struct {
		Location  *utils.Location
		ToExclude map[string]struct{}
	}
)
