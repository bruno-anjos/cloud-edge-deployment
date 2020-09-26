package deployer

import "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"

type (
	GetDeploymentsResponseBody            = []string
	WhoAreYouResponseBody                 = string
	GetHierarchyTableResponseBody         = map[string]*HierarchyEntryDTO
	ResolveInArchimedesResponseBody       = archimedes.ResolvedDTO
	ResolveUpTheTreeResponseBody          = archimedes.ResolvedDTO
	RedirectClientDownTheTreeResponseBody = string
)
