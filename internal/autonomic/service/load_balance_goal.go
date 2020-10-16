package service

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/metrics"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	publicUtils "github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
)

const (
	maximumLoad            = 0.7
	equivalentLoadDiff     = 0.20
	staleCyclesNumToRemove = 5

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

type loadBalanceGoal struct {
	service      *Service
	dependencies []string
	staleCycles  int
}

func newLoadBalanceGoal(service *Service) *loadBalanceGoal {
	dependencies := []string{
		metrics.GetAggLoadPerServiceInChildrenMetricId(service.ServiceId),
		metrics.GetLoadPerServiceInChildrenMetricId(service.ServiceId),
	}

	return &loadBalanceGoal{
		service:      service,
		dependencies: dependencies,
	}
}

func (l *loadBalanceGoal) Optimize(optDomain Domain) (isAlreadyMax bool, optRange Range,
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
		l.staleCycles = 0
		actionArgs = make([]interface{}, lbNumArgs)
		actionArgs[lbActionTypeArgIndex] = actions.RedirectClientsId
		origin := ordered[len(ordered)-1]
		actionArgs[lbFromIndex] = origin
		actionArgs[lbAmountIndex] = int(sortingCriteria[origin].(float64) / 4)
	}
	// else {
	// 	remove := l.checkIfShouldBeRemoved()
	// 	if remove {
	// 		actionArgs = make([]interface{}, lbNumArgs)
	// 		actionArgs[lbActionTypeArgIndex] = actions.RemoveServiceId
	// 	}
	// }

	return
}

func (l *loadBalanceGoal) GenerateDomain(_ interface{}) (domain Domain, info map[string]interface{}, success bool) {
	domain = nil
	info = nil
	success = false

	value, ok := l.service.Environment.GetMetric(metrics.MetricLocationInVicinity)
	if !ok {
		log.Debugf("no value for metric %s", metrics.MetricLocationInVicinity)
		return nil, nil, false
	}

	var locationsInVicinity map[string]publicUtils.Location
	err := mapstructure.Decode(value, &locationsInVicinity)
	if err != nil {
		panic(err)
	}

	value, ok = l.service.Environment.GetMetric(metrics.MetricNodeAddr)
	if !ok {
		log.Debugf("no value for metric %s", metrics.MetricNodeAddr)
		return nil, nil, false
	}

	numChildren := 0
	l.service.Children.Range(func(key, value interface{}) bool {
		numChildren++
		return true
	})

	myself := value.(string)
	info = map[string]interface{}{}
	archClient := archimedes.NewArchimedesClient("")

	for nodeId := range locationsInVicinity {
		_, okS := l.service.Suspected.Load(nodeId)
		if okS || nodeId == myself || nodeId == l.service.ParentId {
			log.Debugf("ignoring %s", nodeId)
			continue
		}
		domain = append(domain, nodeId)
		archClient.SetHostPort(nodeId + ":" + strconv.Itoa(archimedes.Port))
		load, status := archClient.GetLoad(l.service.ServiceId)
		if status != http.StatusOK || numChildren == 0 {
			info[nodeId] = 0.
		} else {
			info[nodeId] = load
		}
		log.Debugf("%s has load: %f(%f/%d)", nodeId, info[nodeId], load, numChildren)
	}

	success = true

	return
}

func (l *loadBalanceGoal) Order(candidates Domain, sortingCriteria map[string]interface{}) (ordered Range) {
	ordered = candidates
	sort.Slice(ordered, func(i, j int) bool {
		loadI := sortingCriteria[ordered[i]].(float64)
		loadJ := sortingCriteria[ordered[j]].(float64)
		return loadI < loadJ
	})

	return
}

func (l *loadBalanceGoal) Filter(candidates, domain Domain) (filtered Range) {
	return DefaultFilter(candidates, domain)
}

func (l *loadBalanceGoal) Cutoff(candidates Domain, candidatesCriteria map[string]interface{}) (cutoff Range,
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

func (l *loadBalanceGoal) GenerateAction(target string, args ...interface{}) actions.Action {
	log.Debugf("generating action %s", (args[lbActionTypeArgIndex]).(string))

	switch args[lbActionTypeArgIndex].(string) {
	case actions.AddServiceId:
		return actions.NewAddServiceAction(l.service.ServiceId, target, false)
	case actions.RedirectClientsId:
		return actions.NewRedirectAction(l.service.ServiceId, args[lbFromIndex].(string), target, args[lbAmountIndex].(int))
	}

	return nil
}

func (l *loadBalanceGoal) TestDryRun() bool {
	return true
}

func (l *loadBalanceGoal) GetDependencies() (metrics []string) {
	return l.dependencies
}

func (l *loadBalanceGoal) increaseMigrationGroupSize() {
	migrationGroupSize *= 2
}

func (l *loadBalanceGoal) decreaseMigrationGroupSize() {
	migrationGroupSize /= 2.0
}

func (l *loadBalanceGoal) resetMigrationGroupSize() {
	migrationGroupSize = defaultGroupSize
}

func (l *loadBalanceGoal) GetId() string {
	return loadBalanceGoalId
}

func (l *loadBalanceGoal) getHighLoads() map[string]float64 {
	highLoads := map[string]float64{}
	l.service.Children.Range(func(key, value interface{}) bool {
		childId := key.(serviceChildrenMapKey)
		metric := metrics.GetLoadPerServiceInChildMetricId(l.service.ServiceId, childId)
		value, mOk := l.service.Environment.GetMetric(metric)
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

func (l *loadBalanceGoal) getAlternativeForHighLoad(highLoads map[string]float64, candidates Range) (alternative string,
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

func (l *loadBalanceGoal) handleOverload(candidates Range) (actionArgs []interface{}, newOptRange Range) {
	actionArgs = make([]interface{}, lbNumArgs, lbNumArgs)
	actionArgs[lbActionTypeArgIndex] = actions.AddServiceId
	deplClient := deployer.NewDeployerClient("")
	for _, candidate := range candidates {
		_, okC := l.service.Children.Load(candidate)
		if !okC {
			deplClient.SetHostPort(candidate + ":" + strconv.Itoa(deployer.Port))
			hasService, _ := deplClient.HasService(l.service.ServiceId)
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

func (l *loadBalanceGoal) checkIfShouldBeRemoved() bool {
	hasChildren := false
	l.service.Children.Range(func(key, value interface{}) bool {
		hasChildren = true
		return false
	})

	if hasChildren {
		return false
	}

	archClient := archimedes.NewArchimedesClient("localhost:" + strconv.Itoa(archimedes.Port))
	load, status := archClient.GetLoad(l.service.ServiceId)
	if status != http.StatusOK {
		log.Errorf("got status %d when asking for load for service %s", status, l.service.ServiceId)
		return false
	}

	if load > 0 {
		return false
	}

	l.staleCycles++

	return l.staleCycles == staleCyclesNumToRemove
}
