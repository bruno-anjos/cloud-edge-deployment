package archimedes

import (
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
)

type (
	GetAllServicesResponseBody     = map[string]*archimedes.Service
	GetServiceInstanceResponseBody = archimedes.Instance
	GetInstanceResponseBody        = archimedes.Instance
	GetServicesTableResponseBody   = archimedes.DiscoverMsg
	GetServiceResponseBody         = map[string]*archimedes.Instance
	ResolveResponseBody            = archimedes.ResolvedDTO
	WhoAreYouResponseBody          = string
)
