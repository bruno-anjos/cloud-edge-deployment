package autonomic

import (
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
)

type (
	GetAllServicesResponseBody = map[string]*ServiceDTO
	ClosestNodeResponseBody    = string
	GetVicinityResponseBody    = map[string]interface{}
	GetMyLocationResponseBody  = *utils.Location
)
