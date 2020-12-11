package deployment

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
)

type (
	domain = []*utils.Node
	result = []*utils.Node
)

type goal interface {
	Optimize(optDomain domain) (isAlreadyMax bool, optRange result, actionArgs []interface{})
	GenerateAction(targets result, args ...interface{}) actions.Action
	GenerateDomain(arg interface{}) (domain domain, info map[string]interface{}, success bool)
	Order(candidates domain, sortingCriteria map[string]interface{}) (ordered result)
	Filter(candidates, domain domain) (filtered result)
	Cutoff(candidates domain, candidatesCriteria map[string]interface{}) (cutoff result, maxed bool)
	GetDependencies() (metrics []string)
	GetID() string
}

func defaultFilter(candidates, domain domain) (filtered result) {
	if domain == nil {
		filtered = candidates

		return
	}

	mappedCandidates := map[string]struct{}{}
	for _, d := range candidates {
		mappedCandidates[d.ID] = struct{}{}
	}

	for _, node := range domain {
		if _, ok := mappedCandidates[node.ID]; ok {
			filtered = append(filtered, node)
		}
	}

	return filtered
}
