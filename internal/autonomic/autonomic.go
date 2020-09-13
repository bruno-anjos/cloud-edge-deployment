package autonomic

import (
	"math/rand"
	"sync"
	"time"

	"github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/environment"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/goals/service_goals"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/metrics"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/strategies"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
)

type (
	servicesMapKey   = string
	servicesMapValue = *service
)

const (
	defaultInterval = 30 * time.Second
)

type service struct {
	ServiceId string
	Strategy  strategies.Strategy
	Children  *sync.Map
	ParentId  *string
	Suspected *sync.Map
}

func newService(serviceId, strategyId string, suspected *sync.Map,
	env *environment.Environment) (*service, error) {
	parentId := ""
	s := &service{
		Children:  &sync.Map{},
		ParentId:  &parentId,
		Suspected: suspected,
	}

	var strategy strategies.Strategy
	switch strategyId {
	case strategies.StrategyLoadBalanceId:
		strategy = strategies.NewDefaultLoadBalanceStrategy(serviceId, s.Children, s.Suspected, &(s.ParentId),
			env)
	case strategies.StrategyIdealLatencyId:
		strategy = strategies.NewDefaultIdealLatencyStrategy(serviceId, s.Children, s.Suspected, &(s.ParentId),
			env)
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

func (a *service) addChild(childId string, location float64) {
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
	*a.ParentId = parentId
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
		ParentId:   *a.ParentId,
	}
}

type system struct {
	services  *sync.Map
	env       *environment.Environment
	suspected *sync.Map

	deployerClient   *deployer.Client
	archimedesClient *archimedes.Client
}

func newSystem() *system {
	return &system{
		services:         &sync.Map{},
		env:              environment.NewEnvironment(),
		suspected:        &sync.Map{},
		deployerClient:   deployer.NewDeployerClient(deployer.DefaultHostPort),
		archimedesClient: archimedes.NewArchimedesClient(archimedes.DefaultHostPort),
	}
}

func (a *system) addService(serviceId, strategyId string) error {
	s, err := newService(serviceId, strategyId, a.suspected, a.env)
	if err != nil {
		panic(err)
	}

	a.services.Store(serviceId, s)

	return nil
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

	location := value.(float64)

	a.suspected.Delete(childId)
	s.addChild(childId, location)
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

func (a *system) start() {
	go func() {
		time.Sleep(time.Duration(rand.Intn(9)+1) * time.Second)
		timer := time.NewTimer(defaultInterval)

		for {
			<-timer.C
			a.services.Range(func(key, value interface{}) bool {
				serviceId := key.(string)
				s := value.(servicesMapValue)
				action := s.generateAction()
				if action == nil {
					return true
				}

				log.Debugf("generated action of type %s for service %s", action.GetActionId(), serviceId)
				a.performAction(action)
				return true
			})
			timer.Reset(defaultInterval)
		}
	}()
}

func (a *system) performAction(action actions.Action) {
	switch assertedAction := action.(type) {
	case *actions.RedirectAction:
		assertedAction.Execute(a.archimedesClient)
	case *actions.AddServiceAction:
		assertedAction.Execute(a.deployerClient)
	case *actions.MigrateAction:
		assertedAction.Execute(a.deployerClient)
	default:
		log.Errorf("could not execute action of type %s", action.GetActionId())
	}
}
