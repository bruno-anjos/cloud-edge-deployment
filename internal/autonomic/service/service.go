package service

import (
	"sync"
	"time"

	"github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/environment"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/metrics"
	public "github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	blacklistDuration = 5 * time.Minute
)

type NodeWithLocation struct {
	NodeId   string
	Location *utils.Location
}

type Service struct {
	ServiceId   string
	Strategy    strategy
	Children    *sync.Map
	ParentId    string
	Suspected   *sync.Map
	Environment *environment.Environment
	Blacklist   *sync.Map
}

func New(serviceId, strategyId string, suspected *sync.Map,
	env *environment.Environment) (*Service, error) {
	s := &Service{
		Children:    &sync.Map{},
		ParentId:    "",
		Suspected:   suspected,
		Environment: env,
		ServiceId:   serviceId,
		Blacklist:   &sync.Map{},
	}

	var strat strategy
	switch strategyId {
	case public.StrategyLoadBalanceId:
		strat = newDefaultLoadBalanceStrategy(s)
	case public.StrategyIdealLatencyId:
		strat = newDefaultIdealLatencyStrategy(s)
	default:
		return nil, errors.Errorf("invalid strategy: %s", strategyId)
	}

	dependencies := strat.GetDependencies()
	if dependencies != nil {
		for _, serviceMetric := range dependencies {
			env.TrackMetric(serviceMetric)
		}
	}

	s.Strategy = strat

	return s, nil
}

func (a *Service) AddChild(childId string, location *utils.Location) {
	node := &NodeWithLocation{
		NodeId:   childId,
		Location: location,
	}
	a.Children.Store(childId, node)
}

func (a *Service) RemoveChild(childId string) {
	a.Children.Delete(childId)
}

func (a *Service) AddSuspectedChild(childId string) {
	a.Suspected.Store(childId, nil)
}

func (a *Service) removeSuspectedChild(childId string) {
	a.Suspected.Delete(childId)
}

func (a *Service) SetParent(parentId string) {
	a.ParentId = parentId
}

func (a *Service) GenerateAction() actions.Action {
	return a.Strategy.Optimize()
}

func (a *Service) ToDTO() *autonomic.ServiceDTO {
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

func (a *Service) GetLoad() float64 {
	metric := metrics.GetLoadPerService(a.ServiceId)
	value, ok := a.Environment.GetMetric(metric)
	if !ok {
		log.Debugf("no value for metric %s", metric)
		return 0
	}

	return value.(float64)
}

func (a *Service) BlacklistNode(nodeId string) {
	log.Debugf("blacklisting %s", nodeId)
	a.Blacklist.Store(nodeId, nil)
	go func() {
		blacklistTimer := time.NewTimer(blacklistDuration)
		<-blacklistTimer.C
		log.Debugf("removing %s from blacklist", nodeId)
		a.Blacklist.Delete(nodeId)
	}()
}

func (a *Service) removeFromBlacklist(nodeId string) {
	a.Blacklist.Delete(nodeId)
}
