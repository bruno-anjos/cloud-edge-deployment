package autonomic

import (
	"sync"
	"time"

	autonomic2 "github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/environment"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/pkg/errors"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/strategies"
)

type (
	servicesMapKey   = string
	servicesMapValue = *autonomic2.Service
)

const (
	defaultInterval = 20 * time.Second
)

type system struct {
	services *sync.Map
	env      *environment.Environment

	deployerClient   *deployer.Client
	archimedesClient *archimedes.Client
}

func newSystem() *system {
	return &system{
		services:         &sync.Map{},
		env:              environment.NewEnvironment(),
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

	service := autonomic2.NewAutonomicService(strategy, childrenMap)
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

func (a *system) getServices() (services map[string]*autonomic2.Service) {
	services = map[string]*autonomic2.Service{}

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
