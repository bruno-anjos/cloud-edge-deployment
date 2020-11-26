package deployment

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

type (
	Domain = []*utils.Node
	Range  = []*utils.Node
)

type Goal interface {
	Optimize(optDomain Domain) (isAlreadyMax bool, optRange Range, actionArgs []interface{})
	GenerateAction(targets Range, args ...interface{}) actions.Action
	GenerateDomain(arg interface{}) (domain Domain, info map[string]interface{}, success bool)
	Order(candidates Domain, sortingCriteria map[string]interface{}) (ordered Range)
	Filter(candidates, domain Domain) (filtered Range)
	Cutoff(candidates Domain, candidatesCriteria map[string]interface{}) (cutoff Range, maxed bool)
	GetDependencies() (metrics []string)
	GetId() string
}

func DefaultFilter(candidates, domain Domain) (filtered Range) {
	if domain == nil {
		filtered = candidates
		return
	}

	mappedCandidates := map[string]struct{}{}
	for _, d := range candidates {
		mappedCandidates[d.Id] = struct{}{}
	}

	for _, node := range domain {
		if _, ok := mappedCandidates[node.Id]; ok {
			filtered = append(filtered, node)
		}
	}

	return filtered
}
