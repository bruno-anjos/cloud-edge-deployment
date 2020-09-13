package strategies

import (
	"sync"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/environment"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/goals"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/goals/service_goals"
)

const (
	StrategyLoadBalanceId = "STRATEGY_LOAD_BALANCE"
)

type loadBalanceStrategy struct {
	*basicStrategy
}

func NewDefaultLoadBalanceStrategy(serviceId string, serviceChildren, suspected *sync.Map, parentId **string,
	env *environment.Environment) *loadBalanceStrategy {
	defaultGoals := []goals.Goal{
		service_goals.NewLoadBalance(serviceId, serviceChildren, suspected, parentId, env),
		service_goals.NewIdealLatency(serviceId, serviceChildren, suspected, parentId, env),
	}
	return &loadBalanceStrategy{
		basicStrategy: newBasicStrategy(StrategyLoadBalanceId, defaultGoals),
	}
}
