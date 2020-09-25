package archimedes

type (
	RegisterServiceRequestBody         = serviceDTO
	RegisterServiceInstanceRequestBody = InstanceDTO
	DiscoverRequestBody                = DiscoverMsg
	ResolveRequestBody                 = struct {
		ToResolve    *ToResolveDTO
		DeploymentId string
	}
	ResolveLocallyRequestBody = ToResolveDTO
	RedirectRequestBody       = redirectDTO
	SetResolutionAnswerRequestBody = struct {
		Resolved *ResolvedDTO
		Id string
	}
)
