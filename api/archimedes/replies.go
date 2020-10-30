package archimedes

type (
	GetAllDeploymentsResponseBody     = map[string]*Deployment
	GetDeploymentInstanceResponseBody = Instance
	GetInstanceResponseBody           = Instance
	GetDeploymentsTableResponseBody   = DiscoverMsg
	GetDeploymentResponseBody         = map[string]*Instance
	ResolveResponseBody               = ResolvedDTO
	WhoAreYouResponseBody             = string
)
