package strategies

import (
	"sync"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/goals"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/goals/service_goals"
)

const (
	StrategyLoadBalanceId = "STRATEGY_LOAD_BALANCE"
)

type loadBalanceStrategy struct {
	*BasicStrategy
}

func NewDefaultLoadBalanceStrategy(serviceId string, serviceChildren *sync.Map,
	env *autonomic.Environment) *loadBalanceStrategy {
	defaultGoals := []goals.Goal{
		service_goals.NewLoadBalance(serviceId, env),
		service_goals.NewIdealLatency(serviceId, serviceChildren, env),
	}
	return &loadBalanceStrategy{
		BasicStrategy: NewBasicStrategy(defaultGoals),
	}
}
