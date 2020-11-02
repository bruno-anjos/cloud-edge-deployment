package deployment

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
)

type (
	Domain = []string
	Range  = []string
)

type Goal interface {
	Optimize(optDomain Domain) (isAlreadyMax bool, optRange Range, actionArgs []interface{})
	GenerateAction(targets []string, args ...interface{}) actions.Action
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
		mappedCandidates[d] = struct{}{}
	}

	for _, nodeId := range domain {
		if _, ok := mappedCandidates[nodeId]; ok {
			filtered = append(filtered, nodeId)
		}
	}

	return filtered
}
