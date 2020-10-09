package archimedes

import (
	publicUtils "github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
)

type (
	RegisterServiceRequestBody         = serviceDTO
	RegisterServiceInstanceRequestBody = InstanceDTO
	DiscoverRequestBody                = DiscoverMsg
	ResolveRequestBody                 = struct {
		ToResolve    *ToResolveDTO
		DeploymentId string
		Location     *publicUtils.Location
	}
	ResolveLocallyRequestBody      = ToResolveDTO
	RedirectRequestBody            = redirectDTO
	SetResolutionAnswerRequestBody = struct {
		Resolved *ResolvedDTO
		Id       string
	}
)
