package archimedes

import (
	publicUtils "github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/golang/geo/s2"
)

type (
	RegisterServiceRequestBody         = serviceDTO
	RegisterServiceInstanceRequestBody = InstanceDTO
	DiscoverRequestBody                = DiscoverMsg
	ResolveRequestBody                 = struct {
		ToResolve    *ToResolveDTO
		DeploymentId string
		Location     *publicUtils.Location
		Id           string
	}
	ResolveLocallyRequestBody      = ToResolveDTO
	RedirectRequestBody            = redirectDTO
	SetResolutionAnswerRequestBody = struct {
		Resolved *ResolvedDTO
		Id       string
	}
	SetExploringClientLocationRequestBody = []s2.CellID
)
