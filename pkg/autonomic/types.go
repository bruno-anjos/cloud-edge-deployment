package autonomic

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/strategies"
)

type AutonomicService struct {
	Strategy *strategies.Strategy
}

func NewAutonomicService(strategy *strategies.Strategy) *AutonomicService {
	return &AutonomicService{
		Strategy: strategy,
	}
}

func (a *AutonomicService) GenerateAction() actions.Action {
	return a.Strategy.Optimize()
}
