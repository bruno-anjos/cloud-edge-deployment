package deployer

import (
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
)

type (
	GetDeploymentsResponseBody    = []string
	WhoAreYouResponseBody         = string
	GetHierarchyTableResponseBody = map[string]*deployer.HierarchyEntryDTO
)
