package service_goals

import (
	"net/http"
	"sort"
	"strconv"
	"sync"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/environment"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/goals"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/metrics"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	publicUtils "github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
)

const (
	maximumLoad        = 0.7
	equivalentLoadDiff = 0.20

	lbNumArgs = 3

	defaultGroupSize = 0.10

	loadBalanceGoalId = "GOAL_LOAD_BALANCE"
)

const (
	lbActionTypeArgIndex = iota
	lbFromIndex
	lbAmountIndex
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
	parentId        *string
}

func NewLoadBalance(serviceId string, children, suspected *sync.Map, parentId *string,
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

	overloaded := false
	for childId, value := range sortingCriteria {
		load := value.(float64)
		if load > maximumLoad {
			log.Debugf("%s is overloaded (%f)", childId, load)
			overloaded = true
			break
		}
	}

	if overloaded {
		actionArgs, optRange = l.handleOverload(filtered)
		isAlreadyMax = !(len(optRange) > 0)
	}

	ordered := l.Order(filtered, sortingCriteria)
	log.Debugf("%s ordered result %+v", loadBalanceGoalId, ordered)

	optRange, isAlreadyMax = l.Cutoff(ordered, sortingCriteria)
	log.Debugf("%s cutoff result (%t)%+v", loadBalanceGoalId, isAlreadyMax, optRange)

	if !isAlreadyMax {
		actionArgs = make([]interface{}, lbNumArgs)
		actionArgs[lbActionTypeArgIndex] = actions.RedirectClientsId
		origin := ordered[len(ordered)-1]
		actionArgs[lbFromIndex] = origin
		actionArgs[lbAmountIndex] = int(sortingCriteria[origin].(float64) / 4)
	}

	return
}

func (l *LoadBalance) GenerateDomain(_ interface{}) (domain goals.Domain, info map[string]interface{}, success bool) {
	domain = nil
	info = nil
	success = false

	value, ok := l.environment.GetMetric(metrics.MetricLocationInVicinity)
	if !ok {
		log.Debugf("no value for metric %s", metrics.MetricLocationInVicinity)
		return nil, nil, false
	}

	var locationsInVicinity map[string]publicUtils.Location
	err := mapstructure.Decode(value, &locationsInVicinity)
	if err != nil {
		panic(err)
	}

	value, ok = l.environment.GetMetric(metrics.MetricNodeAddr)
	if !ok {
		log.Debugf("no value for metric %s", metrics.MetricNodeAddr)
		return nil, nil, false
	}

	numChildren := 0
	l.serviceChildren.Range(func(key, value interface{}) bool {
		numChildren++
		return true
	})

	myself := value.(string)
	info = map[string]interface{}{}
	archClient := archimedes.NewArchimedesClient("")

	for nodeId := range locationsInVicinity {
		_, okS := l.suspected.Load(nodeId)
		if okS || nodeId == myself || nodeId == *l.parentId {
			log.Debugf("ignoring %s", nodeId)
			continue
		}
		domain = append(domain, nodeId)
		archClient.SetHostPort(nodeId + ":" + strconv.Itoa(archimedes.Port))
		load, status := archClient.GetLoad(l.serviceId)
		if status != http.StatusOK || numChildren == 0 {
			info[nodeId] = 0.
		} else {
			info[nodeId] = load
		}
		log.Debugf("%s has load: %f(%f/%d)", nodeId, info[nodeId], load, numChildren)
	}

	success = true

	// TODO GET LOAD PER CHILD
	// loadPerServiceInChildren := metrics.GetLoadPerServiceInChildrenMetricId(l.serviceId)
	// value, ok := l.environment.GetMetric(loadPerServiceInChildren)
	// if !ok {
	// 	log.Debugf("no value for metric %s", loadPerServiceInChildren)
	// 	return
	// }
	//
	// info = value.(map[string]interface{})

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
	loadDiff := mostBusyLoad - leastBusyLoad
	maxed = loadDiff < equivalentLoadDiff
	if maxed {
		return
	}

	log.Debugf("difference between %s(%f) and %s(%f) is %f", mostBusy, mostBusyLoad, leastBusy, leastBusyLoad,
		loadDiff)

	for _, candidate := range candidates {
		if candidatesCriteria[candidate].(float64) <= maximumLoad {
			cutoff = append(cutoff, candidate)
		}
	}

	return
}

func (l *LoadBalance) GenerateAction(target string, args ...interface{}) actions.Action {
	log.Debugf("generating action %s", (args[lbActionTypeArgIndex]).(string))

	switch args[lbActionTypeArgIndex].(string) {
	case actions.AddServiceId:
		return actions.NewAddServiceAction(l.serviceId, target, false)
	case actions.RedirectClientsId:
		return actions.NewRedirectAction(l.serviceId, args[lbFromIndex].(string), target, args[lbAmountIndex].(int))
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

func (l *LoadBalance) handleOverload(candidates goals.Range) (actionArgs []interface{}, newOptRange goals.Range) {
	actionArgs = make([]interface{}, lbNumArgs, lbNumArgs)
	actionArgs[lbActionTypeArgIndex] = actions.AddServiceId
	deplClient := deployer.NewDeployerClient("")
	for _, candidate := range candidates {
		_, okC := l.serviceChildren.Load(candidate)
		if !okC {
			deplClient.SetHostPort(candidate + ":" + strconv.Itoa(deployer.Port))
			hasService, _ := deplClient.HasService(l.serviceId)
			if hasService {
				continue
			}
			newOptRange = append(newOptRange, candidate)
			break
		}
	}

	log.Debugf("%s new opt range %+v", loadBalanceGoalId, newOptRange)

	return
}
