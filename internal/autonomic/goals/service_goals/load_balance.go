package service_goals

import (
	"sort"
	"sync"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/environment"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/goals"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/metrics"
	log "github.com/sirupsen/logrus"
)

const (
	maximumLoad        = 0.8
	equivalentLoadDiff = 0.20

	lbNumArgs         = 1
	actionArgMostBusy = 0

	defaultGroupSize = 0.10

	loadBalanceGoalId = "GOAL_LOAD_BALANCE"
)

var (
	migrationGroupSize = defaultGroupSize
)

type LoadBalance struct {
	serviceId       string
	serviceChildren *sync.Map
	suspected       *sync.Map
	environment     *environment.Environment
	dependencies    []string
	parentId        **string
}

func NewLoadBalance(serviceId string, children, suspected *sync.Map, parentId **string,
	env *environment.Environment) *LoadBalance {
	dependencies := []string{
		metrics.GetAggLoadPerServiceInChildrenMetricId(serviceId),
		metrics.GetLoadPerServiceInChildrenMetricId(serviceId),
	}

	return &LoadBalance{
		serviceId:       serviceId,
		serviceChildren: children,
		suspected:       suspected,
		environment:     env,
		dependencies:    dependencies,
		parentId:        parentId,
	}
}

func (l *LoadBalance) Optimize(optDomain goals.Domain) (isAlreadyMax bool, optRange goals.Range,
	actionArgs []interface{}) {
	isAlreadyMax = true
	optRange = optDomain
	actionArgs = nil

	candidateIds, sortingCriteria, ok := l.GenerateDomain(nil)
	if !ok {
		return
	}
	log.Debugf("%s generated domain %+v", loadBalanceGoalId, candidateIds)

	filtered := l.Filter(candidateIds, optDomain)
	log.Debugf("%s filtered result %+v", loadBalanceGoalId, filtered)

	ordered := l.Order(filtered, sortingCriteria)
	log.Debugf("%s ordered result %+v", loadBalanceGoalId, ordered)

	if len(ordered) < 2 {
		return
	}

	mostBusy := ordered[len(ordered)-1]

	optRange, isAlreadyMax = l.Cutoff(ordered, sortingCriteria)
	log.Debugf("%s cutoff result %+v", loadBalanceGoalId, optRange)

	actionArgs = make([]interface{}, lbNumArgs, lbNumArgs)
	actionArgs[actionArgMostBusy] = mostBusy

	return
}

func (l *LoadBalance) GenerateDomain(_ interface{}) (domain goals.Domain, info map[string]interface{}, success bool) {
	domain = nil
	info = nil
	success = false

	// TODO GET LOAD PER CHILD
	loadPerServiceInChildren := metrics.GetLoadPerServiceInChildrenMetricId(l.serviceId)
	value, ok := l.environment.GetMetric(loadPerServiceInChildren)
	if !ok {
		log.Debugf("no value for metric %s", loadPerServiceInChildren)
		return
	}

	info = value.(map[string]interface{})

	value, ok = l.environment.GetMetric(metrics.MetricNodeAddr)
	if !ok {
		log.Debugf("no value for metric %s", metrics.MetricNodeAddr)
		return nil, nil, false
	}

	myself := value.(string)
	for nodeId := range info {
		_, okC := l.serviceChildren.Load(nodeId)
		_, okS := l.suspected.Load(nodeId)
		if okC || okS || nodeId == myself || nodeId == **l.parentId {
			log.Debugf("ignoring %s", nodeId)
			continue
		}
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
	return l.dependencies
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

func (l *LoadBalance) GetId() string {
	return loadBalanceGoalId
}
