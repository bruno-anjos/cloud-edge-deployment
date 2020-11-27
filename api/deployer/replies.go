package deployer

import (
	"github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

type (
	GetDeploymentsResponseBody            = []string
	WhoAreYouResponseBody                 = string
	GetHierarchyTableResponseBody         = map[string]*HierarchyEntryDTO
	ResolveInArchimedesResponseBody       = archimedes.ResolvedDTO
	ResolveUpTheTreeResponseBody          = archimedes.ResolvedDTO
	RedirectClientDownTheTreeResponseBody = string
	GetFallbackResponseBody               = utils.Node
	IAmYourChildResponseBody              = utils.Node
)
