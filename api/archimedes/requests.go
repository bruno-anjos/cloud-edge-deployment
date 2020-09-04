package archimedes

import (
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
)

type (
	RegisterServiceRequestBody         = archimedes.ServiceDTO
	RegisterServiceInstanceRequestBody = archimedes.InstanceDTO
	DiscoverRequestBody                = archimedes.DiscoverMsg
	ResolveRequestBody                 = archimedes.ToResolveDTO
	RedirectRequestBody                = archimedes.RedirectDTO
)
