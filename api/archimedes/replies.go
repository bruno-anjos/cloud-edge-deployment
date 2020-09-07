package archimedes

type (
	GetAllServicesResponseBody     = map[string]*Service
	GetServiceInstanceResponseBody = Instance
	GetInstanceResponseBody        = Instance
	GetServicesTableResponseBody   = DiscoverMsg
	GetServiceResponseBody         = map[string]*Instance
	ResolveResponseBody            = ResolvedDTO
	WhoAreYouResponseBody          = string
)
