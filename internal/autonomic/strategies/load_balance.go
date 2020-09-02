package strategies

import (
	"time"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/goals"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/goals/service_goals"
)

const (
	STRATEGY_LOAD_BALANCE_ID = "STRATEGY_LOAD_BALANCE"

	defaultLoadBalanceInterval = 20 * time.Second
)

func NewDefaultLoadBalanceStrategy(env *autonomic.Environment) *Strategy {
	defaultGoals := []goals.Goal{service_goals.NewLoadBalance(env), service_goals.NewIdealLatency(env)}
	return NewStrategy(defaultGoals)
}
