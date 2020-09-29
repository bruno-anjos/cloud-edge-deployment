package archimedes

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

type (
	RegisterServiceRequestBody         = serviceDTO
	RegisterServiceInstanceRequestBody = InstanceDTO
	DiscoverRequestBody                = DiscoverMsg
	ResolveRequestBody                 = struct {
		ToResolve    *ToResolveDTO
		DeploymentId string
		Location     *utils.Location
	}
	ResolveLocallyRequestBody      = ToResolveDTO
	RedirectRequestBody            = redirectDTO
	SetResolutionAnswerRequestBody = struct {
		Resolved *ResolvedDTO
		Id       string
	}
)
