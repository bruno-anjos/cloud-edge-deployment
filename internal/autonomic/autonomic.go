package autonomic

import (
	"sync"
	"time"

	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/pkg/errors"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/constraints"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/strategies"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	log "github.com/sirupsen/logrus"
)

type Environment struct {
	trackedMetrics *sync.Map
	metrics        *sync.Map
	constraints    []constraints.Constraint
}

func NewEnvironment() *Environment {
	return &Environment{
		trackedMetrics: &sync.Map{},
		metrics:        &sync.Map{},
		constraints:    []constraints.Constraint{},
	}
}

func (e *Environment) TrackMetric(metricId string) {
	_, loaded := e.trackedMetrics.LoadOrStore(metricId, nil)
	if loaded {
		return
	}

	registerMetricInLowerApi(metricId)
}

func (e *Environment) GetMetric(metricId string) (value interface{}, ok bool) {
	return e.metrics.Load(metricId)
}

func (e *Environment) SetMetric(metricId string, value interface{}) {
	e.metrics.Store(metricId, value)
}

func (e *Environment) DeleteMetric(metricId string) {
	e.metrics.Delete(metricId)
}

func (e *Environment) AddConstraint(constraint constraints.Constraint) {
	e.constraints = append(e.constraints, constraint)
}

func (e *Environment) Copy() (copy *Environment) {
	newMap := &sync.Map{}
	copy = &Environment{metrics: newMap}

	e.metrics.Range(func(key, value interface{}) bool {
		newMap.Store(key, value)
		return true
	})

	return
}

func (e *Environment) CheckConstraints() (invalidConstraints []constraints.Constraint) {
	for _, constraint := range e.constraints {
		metricId := constraint.MetricId()
		value, ok := e.GetMetric(metricId)
		if !ok {
			log.Debugf("metric %s is empty", metricId)
			continue
		}

		valid := constraint.Validate(value)
		if !valid {
			invalidConstraints = append(invalidConstraints, constraint)
		}
	}

	return
}

// TODO change this for lower API call
func registerMetricInLowerApi(metricId string) {

}

type (
	servicesMapKey   = string
	servicesMapValue = *autonomic.Service
)

const (
	defaultInterval = 20 * time.Second
)

type system struct {
	services *sync.Map
	env      *Environment

	deployerClient   *deployer.Client
	archimedesClient *archimedes.Client
}

func newSystem() *system {
	return &system{
		services:         &sync.Map{},
		env:              NewEnvironment(),
		deployerClient:   deployer.NewDeployerClient(deployer.DeployerServiceName),
		archimedesClient: archimedes.NewArchimedesClient(archimedes.ArchimedesServiceName),
	}
}

func (a *system) addService(serviceId, strategyId string) error {
	childrenMap := &sync.Map{}

	var strategy strategies.Strategy
	switch strategyId {
	case strategies.StrategyLoadBalanceId:
		strategy = strategies.NewDefaultLoadBalanceStrategy(serviceId, childrenMap, a.env)
	case strategies.StrategyIdealLatencyId:
		strategy = strategies.NewDefaultIdealLatencyStrategy(serviceId, childrenMap, a.env)
	default:
		return errors.Errorf("invalid strategy: %s", strategyId)
	}

	dependencies := strategy.GetDependencies()
	if dependencies != nil {
		for _, serviceMetric := range dependencies {
			a.env.TrackMetric(serviceMetric)
		}
	}

	service := autonomic.NewAutonomicService(strategy, childrenMap)
	a.services.Store(serviceId, service)

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

	service := value.(servicesMapValue)
	service.AddChild(childId)
}

func (a *system) removeServiceChild(serviceId, childId string) {
	value, ok := a.services.Load(serviceId)
	if !ok {
		return
	}

	service := value.(servicesMapValue)
	service.RemoveChild(childId)
}

func (a *system) getServices() (services map[string]*autonomic.Service) {
	services = map[string]*autonomic.Service{}

	a.services.Range(func(key, value interface{}) bool {
		serviceId := key.(servicesMapKey)
		service := value.(servicesMapValue)

		services[serviceId] = service

		return true
	})

	return
}

func (a *system) start() {
	go func() {
		timer := time.NewTimer(defaultInterval)

		for {
			<-timer.C
			a.services.Range(func(key, value interface{}) bool {
				service := value.(servicesMapValue)
				action := service.GenerateAction()
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
	case *actions.MigrateAction:
		assertedAction.Execute(a.deployerClient)
	}
}
