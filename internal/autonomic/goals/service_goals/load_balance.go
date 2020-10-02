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
	maximumLoad        = 0.7
	equivalentLoadDiff = 0.20

	lbNumArgs = 2

	defaultGroupSize = 0.10

	loadBalanceGoalId = "GOAL_LOAD_BALANCE"
)

const (
	lbActionTypeArgIndex = iota
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

	overloaded := false
	for _, value := range sortingCriteria {
		load := value.(float64)
		if load > maximumLoad {
			overloaded = true
			break
		}
	}

	actionArgs = make([]interface{}, lbNumArgs, lbNumArgs)
	if overloaded {
		actionArgs[lbActionTypeArgIndex] = actions.AddServiceId
		optRange = goals.Range{}
		for _, candidateFromPreviousGoal := range optDomain {
			_, cOK := l.serviceChildren.Load(candidateFromPreviousGoal)
			if !cOK {
				optRange = append(optRange, candidateFromPreviousGoal)
				break
			}
		}

		isAlreadyMax = false
		return
	}

	filtered := l.Filter(candidateIds, optDomain)
	log.Debugf("%s filtered result %+v", loadBalanceGoalId, filtered)

	ordered := l.Order(filtered, sortingCriteria)
	log.Debugf("%s ordered result %+v", loadBalanceGoalId, ordered)

	optRange, isAlreadyMax = l.Cutoff(ordered, sortingCriteria)
	log.Debugf("%s cutoff result %+v", loadBalanceGoalId, optRange)

	// TODO understand where migrate action fits
	// if furthestChild != "" {
	// 	actionArgs[ilActionTypeArgIndex] = actions.MigrateServiceId
	// 	actionArgs[ilFromIndex] = furthestChild
	// } else {
	// 	isAlreadyMax = true
	// }

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

	if len(candidates) < 2 {
		return
	}

	leastBusy := candidates[0]
	mostBusy := candidates[len(candidates)-1]

	mostBusyLoad := candidatesCriteria[mostBusy].(float64)
	leastBusyLoad := candidatesCriteria[leastBusy].(float64)

	if mostBusyLoad-leastBusyLoad < equivalentLoadDiff {
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
	log.Debugf("generating action %s", (args[lbActionTypeArgIndex]).(string))

	switch args[ilActionTypeArgIndex].(string) {
	case actions.AddServiceId:
		return actions.NewAddServiceAction(l.serviceId, target)
	}

	return nil
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

func (l *LoadBalance) getHighLoads() map[string]float64 {
	highLoads := map[string]float64{}
	l.serviceChildren.Range(func(key, value interface{}) bool {
		childId := key.(serviceChildrenMapKey)
		metric := metrics.GetLoadPerServiceInChildMetricId(l.serviceId, childId)
		value, mOk := l.environment.GetMetric(metric)
		if !mOk {
			log.Debugf("no value for metric %s", metric)
			return true
		}

		load := value.(float64)
		if load > maximumLoad {
			highLoads[childId] = load
		}

		return true
	})

	return highLoads
}

func (l *LoadBalance) getAlternativeForHighLoad(highLoads map[string]float64, candidates goals.Range) (alternative string,
	ok bool) {
	var highLoadChildIds []string
	for childId := range highLoads {
		highLoadChildIds = append(highLoadChildIds, childId)
	}

	log.Debugf("children with high loads: %v", highLoadChildIds)

	var filteredCandidates []string
	for _, candidate := range candidates {
		_, ok = highLoads[candidate]
		if !ok {
			filteredCandidates = append(filteredCandidates, candidate)
		}
	}

	if len(filteredCandidates) == 0 {
		ok = false
		return
	}

	alternative = filteredCandidates[0]
	return
}
