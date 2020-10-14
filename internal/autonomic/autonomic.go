package autonomic

import (
	"math"
	"sort"
	"sync"
	"time"

	"github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/environment"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/goals/service_goals"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/metrics"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/strategies"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
)

type (
	servicesMapKey   = string
	servicesMapValue = *service
)

const (
	defaultInterval   = 1 * time.Minute
	blacklistDuration = 5 * time.Minute
)

type service struct {
	ServiceId   string
	Strategy    strategies.Strategy
	Children    *sync.Map
	ParentId    string
	Suspected   *sync.Map
	Environment *environment.Environment
	Blacklist   *sync.Map
}

func newService(serviceId, strategyId string, suspected *sync.Map,
	env *environment.Environment) (*service, error) {
	s := &service{
		Children:    &sync.Map{},
		ParentId:    "",
		Suspected:   suspected,
		Environment: env,
		ServiceId:   serviceId,
		Blacklist:   &sync.Map{},
	}

	var strategy strategies.Strategy
	switch strategyId {
	case strategies.StrategyLoadBalanceId:
		strategy = strategies.NewDefaultLoadBalanceStrategy(serviceId, s.Children, s.Suspected, &(s.ParentId),
			env, s.Blacklist)
	case strategies.StrategyIdealLatencyId:
		strategy = strategies.NewDefaultIdealLatencyStrategy(serviceId, s.Children, s.Suspected, &(s.ParentId),
			env, s.Blacklist)
	default:
		return nil, errors.Errorf("invalid strategy: %s", strategyId)
	}

	dependencies := strategy.GetDependencies()
	if dependencies != nil {
		for _, serviceMetric := range dependencies {
			env.TrackMetric(serviceMetric)
		}
	}

	s.Strategy = strategy

	return s, nil
}

func (a *service) addChild(childId string, location *utils.Location) {
	node := &service_goals.NodeWithLocation{
		NodeId:   childId,
		Location: location,
	}
	a.Children.Store(childId, node)
}

func (a *service) removeChild(childId string) {
	a.Children.Delete(childId)
}

func (a *service) addSuspectedChild(childId string) {
	a.Suspected.Store(childId, nil)
}

func (a *service) removeSuspectedChild(childId string) {
	a.Suspected.Delete(childId)
}

func (a *service) setParent(parentId string) {
	a.ParentId = parentId
}

func (a *service) generateAction() actions.Action {
	return a.Strategy.Optimize()
}

func (a *service) toDTO() *autonomic.ServiceDTO {
	var children []string
	a.Children.Range(func(key, value interface{}) bool {
		childId := key.(string)
		children = append(children, childId)
		return true
	})

	return &autonomic.ServiceDTO{
		ServiceId:  a.ServiceId,
		StrategyId: a.Strategy.GetId(),
		Children:   children,
		ParentId:   a.ParentId,
	}
}

func (a *service) getLoad() float64 {
	metric := metrics.GetLoadPerService(a.ServiceId)
	value, ok := a.Environment.GetMetric(metric)
	if !ok {
		log.Debugf("no value for metric %s", metric)
		return 0
	}

	return value.(float64)
}

func (a *service) blacklistNode(nodeId string) {
	log.Debugf("blacklisting %s", nodeId)
	a.Blacklist.Store(nodeId, nil)
	go func() {
		blacklistTimer := time.NewTimer(blacklistDuration)
		<-blacklistTimer.C
		log.Debugf("removing %s from blacklist", nodeId)
		a.Blacklist.Delete(nodeId)
	}()
}

func (a *service) removeFromBlacklist(nodeId string) {
	a.Blacklist.Delete(nodeId)
}

type (
	system struct {
		services  *sync.Map
		env       *environment.Environment
		suspected *sync.Map

		deployerClient   *deployer.Client
		archimedesClient *archimedes.Client
		exploring        sync.Map
	}

	exploringMapValue = chan struct{}
)

func newSystem() *system {
	return &system{
		services:         &sync.Map{},
		env:              environment.NewEnvironment(),
		suspected:        &sync.Map{},
		deployerClient:   deployer.NewDeployerClient(deployer.DefaultHostPort),
		archimedesClient: archimedes.NewArchimedesClient(archimedes.DefaultHostPort),
		exploring:        sync.Map{},
	}
}

func (a *system) addService(serviceId, strategyId string) {
	if _, ok := a.services.Load(serviceId); ok {
		return
	}

	s, err := newService(serviceId, strategyId, a.suspected, a.env)
	if err != nil {
		panic(err)
	}

	a.services.Store(serviceId, s)
	go a.handleService(s)

	return
}

func (a *system) removeService(serviceId string) {
	a.services.Delete(serviceId)
}

func (a *system) addServiceChild(serviceId, childId string) {
	value, ok := a.services.Load(serviceId)
	if !ok {
		return
	}

	s := value.(servicesMapValue)
	value, ok = a.env.GetMetric(metrics.MetricLocationInVicinity)
	if !ok {
		log.Errorf("no metric %s", metrics.MetricLocationInVicinity)
		return
	}

	locations := value.(map[string]interface{})
	value, ok = locations[childId]
	if !ok {
		log.Errorf("no location for child %s", childId)
		return
	}

	var location utils.Location
	err := mapstructure.Decode(value, &location)
	if err != nil {
		panic(err)
	}

	a.suspected.Delete(childId)
	s.addChild(childId, &location)
}

func (a *system) removeServiceChild(serviceId, childId string) {
	value, ok := a.services.Load(serviceId)
	if !ok {
		return
	}

	s := value.(servicesMapValue)
	s.addSuspectedChild(childId)
	s.removeChild(childId)
}

func (a *system) setServiceParent(serviceId, parentId string) {
	value, ok := a.services.Load(serviceId)
	if !ok {
		return
	}

	s := value.(servicesMapValue)
	s.setParent(parentId)
}

func (a *system) getServices() (services map[string]*service) {
	services = map[string]*service{}

	a.services.Range(func(key, value interface{}) bool {
		serviceId := key.(servicesMapKey)
		s := value.(servicesMapValue)

		services[serviceId] = s

		return true
	})

	return
}

func (a *system) isNodeInVicinity(nodeId string) bool {
	value, ok := a.env.GetMetric(metrics.MetricLocationInVicinity)
	if !ok {
		return false
	}

	vicinity := value.(map[string]interface{})
	_, ok = vicinity[nodeId]

	return ok
}

func (a *system) closestNodeTo(location *utils.Location, toExclude map[string]struct{}) (nodeId string) {
	value, ok := a.env.GetMetric(metrics.MetricLocationInVicinity)
	if !ok {
		return ""
	}

	vicinity := value.(map[string]interface{})
	var ordered []string

	for node := range vicinity {
		if _, ok = toExclude[node]; ok {
			continue
		}
		ordered = append(ordered, node)
	}

	sort.Slice(ordered, func(i, j int) bool {
		var iLoc utils.Location
		err := mapstructure.Decode(vicinity[ordered[i]], &iLoc)
		if err != nil {
			panic(err)
		}

		var jLoc utils.Location
		err = mapstructure.Decode(vicinity[ordered[j]], &jLoc)
		if err != nil {
			panic(err)
		}

		return math.Abs(iLoc.CalcDist(location)) < math.Abs(jLoc.CalcDist(location))
	})

	if len(ordered) < 1 {
		return ""
	}

	return ordered[0]
}

func (a *system) getVicinity() map[string]interface{} {
	value, ok := a.env.GetMetric(metrics.MetricLocationInVicinity)
	if !ok {
		return nil
	}

	vicinity := value.(map[string]interface{})

	return vicinity
}

func (a *system) getMyLocation() *utils.Location {
	value, ok := a.env.GetMetric(metrics.MetricLocation)
	if !ok {
		return nil
	}

	var location utils.Location
	err := mapstructure.Decode(value, &location)
	if err != nil {
		panic(err)
	}

	return &location
}

func (a *system) handleService(service *service) {
	timer := time.NewTimer(defaultInterval)

	for {
		<-timer.C

		log.Debugf("evaluating service %s", service.ServiceId)

		action := service.generateAction()
		if action != nil {
			log.Debugf("generated action of type %s for service %s", action.GetActionId(), service.ServiceId)
			a.performAction(action)
		}
		timer.Reset(defaultInterval)
	}
}

func (a *system) performAction(action actions.Action) {
	switch assertedAction := action.(type) {
	case *actions.RedirectAction:
		assertedAction.Execute(a.archimedesClient)
	case *actions.AddServiceAction:
		if assertedAction.Exploring {
			id := assertedAction.GetServiceId() + "_" + assertedAction.GetTarget()
			exploreChan := make(chan interface{})
			a.exploring.Store(id, exploreChan)
			go a.waitToBlacklist(assertedAction.GetServiceId(), assertedAction.GetTarget(), exploreChan)
		}
		assertedAction.Execute(a.deployerClient)
	case *actions.MigrateAction:
		assertedAction.Execute(a.deployerClient)
	default:
		log.Errorf("could not execute action of type %s", action.GetActionId())
	}
}

func (a *system) getLoad(serviceId string) (float64, bool) {
	value, ok := a.services.Load(serviceId)
	if !ok {
		return 0, false
	}

	return value.(servicesMapValue).getLoad(), true
}

func (a *system) setExploreSuccess(deploymentId, childId string) bool {
	id := deploymentId + "_" + childId
	value, ok := a.exploring.Load(id)
	if !ok {
		return false
	}

	a.exploring.Delete(id)

	exploreChan := value.(exploringMapValue)
	close(exploreChan)

	return true
}

func (a *system) waitToBlacklist(serviceId, childId string, exploredChan <-chan interface{}) {
	value, ok := a.services.Load(serviceId)
	if !ok {
		return
	}

	auxService := value.(servicesMapValue)

	interval := (4 * 30) * time.Second

	timer := time.NewTimer(interval)
	select {
	case <-exploredChan:
		log.Debugf("exploring %s through %s was a success", serviceId, childId)
		return
	case <-timer.C:
		auxService.blacklistNode(childId)
	}
}
