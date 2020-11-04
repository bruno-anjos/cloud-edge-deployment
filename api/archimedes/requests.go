package archimedes

import (
	"github.com/golang/geo/s2"
)

type (
	RegisterDeploymentRequestBody         = deploymentDTO
	RegisterDeploymentInstanceRequestBody = InstanceDTO
	DiscoverRequestBody                   = DiscoverMsg
	ResolveRequestBody                    = struct {
		ToResolve    *ToResolveDTO
		DeploymentId string
		Location     s2.CellID
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
