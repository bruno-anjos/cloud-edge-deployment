package service_goals

import (
	"sort"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/goals"
	log "github.com/sirupsen/logrus"
)

const (
	maximumLoadPercentage    = 1.1
	equivalentLoadPercentage = 0.8
)

var (
	loadBalanceDependencies = []string{
		autonomic.METRIC_LOAD,
		autonomic.METRIC_LOAD_IN_VICINITY,
	}
)

type LoadBalance struct {
	environment *autonomic.Environment
}

func NewLoadBalance(env *autonomic.Environment) *LoadBalance {
	return &LoadBalance{
		environment: env,
	}
}

func (l *LoadBalance) Optimize(optDomain goals.Domain) (isAlreadyMax bool, optRange goals.Range) {
	isAlreadyMax = true
	optRange = nil

	value, ok := l.environment.GetMetric(autonomic.METRIC_LOAD)
	if !ok {
		log.Debugf("no value for metric %s", autonomic.METRIC_LOAD)
		return
	}

	myLoad := value.(float64)

	candidateIds, candidates, ok := l.GenerateDomain(nil)
	if !ok {
		return
	}

	filtered := l.Filter(candidateIds, optDomain)
	ordered := l.Order(filtered, candidates)

	optRange, isAlreadyMax = l.Cutoff(ordered, myLoad, candidates)

	return
}

func (l *LoadBalance) GenerateDomain(_ interface{}) (domain goals.Domain, info map[string]interface{}, success bool) {
	domain = nil
	info = nil
	success = false

	value, ok := l.environment.GetMetric(autonomic.METRIC_LOAD_IN_VICINITY)
	if !ok {
		log.Debugf("no value for metric %s", autonomic.METRIC_LOAD_IN_VICINITY)
		return
	}

	info = value.(map[string]interface{})
	for nodeId := range info {
		domain = append(domain, nodeId)
	}
	success = true

	return
}

func (l *LoadBalance) Order(candidates goals.Domain, sortingCriteria map[string]interface{}) (ordered goals.Range) {
	ordered = candidates
	sort.Slice(ordered, func(i, j int) bool {
		loadI := sortingCriteria[ordered[i]].(float64)
		loadJ := sortingCriteria[ordered[j]].(float64)

		return loadI < loadJ
	})

	return
}

func (l *LoadBalance) Filter(candidates, domain goals.Domain) (filtered goals.Range) {
	return goals.DefaultFilter(candidates, domain)
}

func (l *LoadBalance) Cutoff(candidates goals.Domain, myCriteria interface{},
	candidatesCriteria map[string]interface{}) (cutoff goals.Range, maxed bool) {
	maxed = true
	for _, candidate := range candidates {
		percentage := candidatesCriteria[candidate].(float64) / myCriteria.(float64)
		if percentage < maximumLoadPercentage {
			cutoff = append(cutoff, candidate)
		}
		if percentage < equivalentLoadPercentage {
			maxed = false
		}
	}

	return
}

func (l *LoadBalance) GenerateAction(target string) actions.Action {
	return actions.NewAddServiceAction(target)
}

func (l *LoadBalance) TestDryRun() bool {
	return true
}

func (l *LoadBalance) GetDependencies() (metrics []string) {
	return loadBalanceDependencies
}
