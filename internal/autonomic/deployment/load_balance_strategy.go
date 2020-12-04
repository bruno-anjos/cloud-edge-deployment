package deployment

import (
	public "github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
)

type loadBalanceStrategy struct {
	*basicStrategy
}

func newDefaultLoadBalanceStrategy(deployment *Deployment) *loadBalanceStrategy {
	defaultGoals := []goal{
		newLoadBalanceGoal(deployment),
		newIdealLatencyGoal(deployment),
	}
	return &loadBalanceStrategy{
		basicStrategy: newBasicStrategy(public.StrategyLoadBalanceId, defaultGoals),
	}
}
