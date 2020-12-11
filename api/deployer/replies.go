package deployer

import (
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
)

type (
	GetDeploymentsResponseBody    = []string
	GetHierarchyTableResponseBody = map[string]*HierarchyEntryDTO
	GetFallbackResponseBody       = utils.Node
)
