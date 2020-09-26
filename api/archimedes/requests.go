package archimedes

type (
	RegisterServiceRequestBody         = serviceDTO
	RegisterServiceInstanceRequestBody = InstanceDTO
	DiscoverRequestBody                = DiscoverMsg
	ResolveRequestBody                 = struct {
		ToResolve    *ToResolveDTO
		DeploymentId string
		Location     float64
	}
	ResolveLocallyRequestBody      = ToResolveDTO
	RedirectRequestBody            = redirectDTO
	SetResolutionAnswerRequestBody = struct {
		Resolved *ResolvedDTO
		Id       string
	}
)
