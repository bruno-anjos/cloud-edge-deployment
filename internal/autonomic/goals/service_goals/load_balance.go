package service_goals

import (
	"sort"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/goals"
	log "github.com/sirupsen/logrus"
)

const (
	maximumLoad        = 0.8
	equivalentLoadDiff = 0.20

	lbNumArgs         = 1
	actionArgMostBusy = 0

	defaultGroupSize = 0.10
)

var (
	loadBalanceDependencies = []string{
		autonomic.METRIC_AGG_LOAD_PER_SERVICE_IN_CHILDREN,
	}
	migrationGroupSize = defaultGroupSize
)

type LoadBalance struct {
	serviceId   string
	environment *autonomic.Environment
}

func NewLoadBalance(serviceId string, env *autonomic.Environment) *LoadBalance {
	return &LoadBalance{
		serviceId:   serviceId,
		environment: env,
	}
}

func (l *LoadBalance) Optimize(optDomain goals.Domain) (isAlreadyMax bool, optRange goals.Range,
	actionArgs []interface{}) {
	isAlreadyMax = true
	optRange = nil
	actionArgs = nil

	candidateIds, sortingCriteria, ok := l.GenerateDomain(nil)
	if !ok {
		return
	}

	filtered := l.Filter(candidateIds, optDomain)
	ordered := l.Order(filtered, sortingCriteria)

	mostBusy := ordered[len(ordered)-1]

	optRange, isAlreadyMax = l.Cutoff(ordered, sortingCriteria)
	actionArgs = make([]interface{}, lbNumArgs, lbNumArgs)
	actionArgs[actionArgMostBusy] = mostBusy

	return
}

func (l *LoadBalance) GenerateDomain(_ interface{}) (domain goals.Domain, info map[string]interface{}, success bool) {
	domain = nil
	info = nil
	success = false

	// TODO GET LOAD PER CHILD
	value, ok := l.environment.GetMetric(autonomic.METRIC_LOAD_PER_SERVICE_IN_CHILD)
	if !ok {
		log.Debugf("no value for metric %s", autonomic.METRIC_LOAD_PER_SERVICE_IN_CHILD)
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

func (l *LoadBalance) Cutoff(candidates goals.Domain, candidatesCriteria map[string]interface{}) (cutoff goals.Range,
	maxed bool) {

	cutoff = nil
	maxed = true

	leastBusy := candidates[0]
	if len(candidates)-1 < 1 {
		return
	}

	mostBusy := candidates[len(candidates)-1]

	if candidatesCriteria[mostBusy].(float64)-candidatesCriteria[leastBusy].(float64) < equivalentLoadDiff {
		cutoff = nil
		maxed = true
	} else {
		for _, candidate := range candidates {
			if candidatesCriteria[candidate].(float64) <= maximumLoad {
				cutoff = append(cutoff, candidate)
			}
		}
		maxed = false
	}

	return
}

func (l *LoadBalance) GenerateAction(target string, args ...interface{}) actions.Action {
	from := args[actionArgMostBusy].(string)
	return actions.NewRedirectAction(l.serviceId, from, target, migrationGroupSize)
}

func (l *LoadBalance) TestDryRun() bool {
	return true
}

func (l *LoadBalance) GetDependencies() (metrics []string) {
	return loadBalanceDependencies
}

func (l *LoadBalance) IncreaseMigrationGroupSize() {
	migrationGroupSize *= 2
}

func (l *LoadBalance) DecreaseMigrationGroupSize() {
	migrationGroupSize /= 2.0
}

func (l *LoadBalance) ResetMigrationGroupSize() {
	migrationGroupSize = defaultGroupSize
}
