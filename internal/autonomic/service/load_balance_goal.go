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
	maximumLoad            = 300
	equivalentLoadDiff     = 2.0
	staleCyclesNumToRemove = 5

	defaultGroupSize = 0.10

	loadBalanceGoalId = "GOAL_LOAD_BALANCE"
)

const (
	lbActionTypeArgIndex = iota
	lbFromIndex
	lbAmountIndex
	lbNumArgs
)

var (
	migrationGroupSize = defaultGroupSize
)

type (
	loadType = int
)

type serviceLoadBalanceGoal struct {
	service      *Service
	dependencies []string
	staleCycles  int
}

func newLoadBalanceGoal(service *Service) *serviceLoadBalanceGoal {
	dependencies := []string{
		metrics.GetAggLoadPerServiceInChildrenMetricId(service.ServiceId),
		metrics.GetLoadPerServiceInChildrenMetricId(service.ServiceId),
	}

	return &serviceLoadBalanceGoal{
		service:      service,
		dependencies: dependencies,
	}
}

func (l *serviceLoadBalanceGoal) Optimize(optDomain Domain) (isAlreadyMax bool, optRange Range,
	actionArgs []interface{}) {
	isAlreadyMax = true
	optRange = optDomain
	actionArgs = nil

	value, ok := l.service.Environment.GetMetric(metrics.MetricNodeAddr)
	if !ok {
		log.Debugf("no value for metric %s", metrics.MetricNodeAddr)
		return
	}

	candidateIds, sortingCriteria, ok := l.GenerateDomain(nil)
	if !ok {
		return
	}
	log.Debugf("%s generated domain %+v", loadBalanceGoalId, candidateIds)

	filtered := l.Filter(candidateIds, optDomain)
	log.Debugf("%s filtered result %+v", loadBalanceGoalId, filtered)

	ordered := l.Order(filtered, sortingCriteria)
	log.Debugf("%s ordered result %+v", loadBalanceGoalId, ordered)

	optRange, isAlreadyMax = l.Cutoff(ordered, sortingCriteria)
	log.Debugf("%s cutoff result (%t)%+v", loadBalanceGoalId, isAlreadyMax, optRange)

	overloaded := false
	myLoad := sortingCriteria[myself.Id].(loadType)

	if myLoad > maximumLoad {
		log.Debugf("im overloaded (%d)", myLoad)
		overloaded = true
	}

	if overloaded {
		var (
			nodeId       string
			alternatives []string
			deplClient   = deployer.NewDeployerClient("")
		)
		for nodeId, value = range sortingCriteria {
			load := value.(loadType)
			if float64(load)+float64(myLoad)/2. < maximumLoad {
				deplClient.SetHostPort(nodeId + ":" + strconv.Itoa(deployer.Port))
				hasService, _ := deplClient.HasService(l.service.ServiceId)
				if hasService {
					alternatives = append(alternatives, nodeId)
				}
			}
		}

		hasAlternatives := len(alternatives) > 0
		log.Debugf("overloaded: %t, alternative: %t", overloaded, hasAlternatives)

		if !hasAlternatives {
			actionArgs, optRange = l.handleOverload(optRange)
			isAlreadyMax = !(len(optRange) > 0)
		}
	} else if !isAlreadyMax {
		l.staleCycles = 0
		actionArgs = make([]interface{}, lbNumArgs)
		actionArgs[lbActionTypeArgIndex] = actions.RedirectClientsId
		origin := ordered[len(ordered)-1]
		actionArgs[lbFromIndex] = origin
		actionArgs[lbAmountIndex] = sortingCriteria[origin].(loadType) / 4
		log.Debugf("will try to achieve load equilibrium redirecting %d clients from %s to %s",
			actionArgs[lbAmountIndex], origin, optRange[0])
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

func (l *serviceLoadBalanceGoal) GenerateDomain(_ interface{}) (domain Domain, info map[string]interface{},
	success bool) {
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

	info = map[string]interface{}{}
	archClient := archimedes.NewArchimedesClient(myself.Id + ":" + strconv.Itoa(archimedes.Port))
	load, status := archClient.GetLoad(l.service.ServiceId)
	if status != http.StatusOK {
		load = 0
	}

	info[myself.Id] = load

	for nodeId := range locationsInVicinity {
		_, okS := l.service.Suspected.Load(nodeId)
		if okS || nodeId == l.service.ParentId {
			log.Debugf("ignoring %s", nodeId)
			continue
		}
		domain = append(domain, nodeId)
		archClient.SetHostPort(nodeId + ":" + strconv.Itoa(archimedes.Port))
		load, status = archClient.GetLoad(l.service.ServiceId)
		if status != http.StatusOK {
			info[nodeId] = 0
		} else {
			info[nodeId] = load
		}
		log.Debugf("%s has load: %d(%d)", nodeId, info[nodeId], load)
	}

	success = true

	return
}

func (l *serviceLoadBalanceGoal) Order(candidates Domain, sortingCriteria map[string]interface{}) (ordered Range) {
	ordered = candidates
	sort.Slice(ordered, func(i, j int) bool {
		loadI := sortingCriteria[ordered[i]].(loadType)
		loadJ := sortingCriteria[ordered[j]].(loadType)
		return loadI < loadJ
	})

	return
}

func (l *serviceLoadBalanceGoal) Filter(candidates, domain Domain) (filtered Range) {
	return DefaultFilter(candidates, domain)
}

func (l *serviceLoadBalanceGoal) Cutoff(candidates Domain, candidatesCriteria map[string]interface{}) (cutoff Range,
	maxed bool) {

	cutoff = nil
	maxed = true

	if len(candidates) < 2 {
		cutoff = candidates
		return
	}

	leastBusy := candidates[0]
	mostBusy := candidates[len(candidates)-1]

	mostBusyLoad := candidatesCriteria[mostBusy].(loadType)
	leastBusyLoad := candidatesCriteria[leastBusy].(loadType)
	loadDiff := 0.
	if leastBusyLoad != 0 {
		loadDiff = float64(mostBusyLoad) / float64(leastBusyLoad)
	}
	maxed = loadDiff < equivalentLoadDiff
	if maxed {
		return
	}

	log.Debugf("difference between %s(%d) and %s(%d) is %f", mostBusy, mostBusyLoad, leastBusy, leastBusyLoad,
		loadDiff)

	for _, candidate := range candidates {
		if candidatesCriteria[candidate].(loadType) <= maximumLoad {
			cutoff = append(cutoff, candidate)
		}
	}

	return
}

func (l *serviceLoadBalanceGoal) GenerateAction(target string, args ...interface{}) actions.Action {
	log.Debugf("generating action %s", (args[lbActionTypeArgIndex]).(string))

	switch args[lbActionTypeArgIndex].(string) {
	case actions.ExtendServiceId:
		return actions.NewExtendServiceAction(l.service.ServiceId, target, false, myself,
			nil)
	case actions.RedirectClientsId:
		return actions.NewRedirectAction(l.service.ServiceId, args[lbFromIndex].(string), target, args[lbAmountIndex].(int))
	}

	return nil
}

func (l *serviceLoadBalanceGoal) TestDryRun() bool {
	return true
}

func (l *serviceLoadBalanceGoal) GetDependencies() (metrics []string) {
	return l.dependencies
}

func (l *serviceLoadBalanceGoal) increaseMigrationGroupSize() {
	migrationGroupSize *= 2
}

func (l *serviceLoadBalanceGoal) decreaseMigrationGroupSize() {
	migrationGroupSize /= 2.0
}

func (l *serviceLoadBalanceGoal) resetMigrationGroupSize() {
	migrationGroupSize = defaultGroupSize
}

func (l *serviceLoadBalanceGoal) GetId() string {
	return loadBalanceGoalId
}

func (l *serviceLoadBalanceGoal) getHighLoads() map[string]loadType {
	highLoads := map[string]loadType{}
	l.service.Children.Range(func(key, value interface{}) bool {
		childId := key.(serviceChildrenMapKey)
		metric := metrics.GetLoadPerServiceInChildMetricId(l.service.ServiceId, childId)
		value, mOk := l.service.Environment.GetMetric(metric)
		if !mOk {
			log.Debugf("no value for metric %s", metric)
			return true
		}

		load := value.(loadType)
		if load > maximumLoad {
			highLoads[childId] = load
		}

		return true
	})

	return highLoads
}

func (l *serviceLoadBalanceGoal) handleOverload(candidates Range) (
	actionArgs []interface{}, newOptRange Range) {
	actionArgs = make([]interface{}, lbNumArgs, lbNumArgs)

	actionArgs[lbActionTypeArgIndex] = actions.ExtendServiceId
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

func (l *serviceLoadBalanceGoal) checkIfShouldBeRemoved() bool {
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
