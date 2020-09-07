package autonomic

import (
	"sync"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/strategies"
)

type Service struct {
	Strategy strategies.Strategy
	Children *sync.Map
}

func NewAutonomicService(strategy strategies.Strategy, childrenMap *sync.Map) *Service {
	return &Service{
		Strategy: strategy,
		Children: childrenMap,
	}
}

func (a *Service) AddChild(childId string) {
	a.Children.Store(childId, struct{}{})
}

func (a *Service) RemoveChild(childId string) {
	a.Children.Delete(childId)
}

func (a *Service) GenerateAction() actions.Action {
	return a.Strategy.Optimize()
}
