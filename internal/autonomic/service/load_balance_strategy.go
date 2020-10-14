package service

import (
	public "github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
)

type loadBalanceStrategy struct {
	*basicStrategy
}

func newDefaultLoadBalanceStrategy(service *Service) *loadBalanceStrategy {
	defaultGoals := []Goal{
		newLoadBalanceGoal(service),
		NewIdealLatencyGoal(service),
	}
	return &loadBalanceStrategy{
		basicStrategy: newBasicStrategy(public.StrategyLoadBalanceId, defaultGoals),
	}
}
