package strategies

import (
	"time"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/goals"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/goals/service_goals"
)

const (
	STRATEGY_IDEAL_LATENCY_ID = "STRATEGY_IDEAL_LATENCY"

	defaultIdealLatencyInterval = 20 * time.Second
)

func NewDefaultIdealLatencyStrategy(env *autonomic.Environment) *Strategy {
	defaultGoals := []goals.Goal{service_goals.NewIdealLatency(env), service_goals.NewLoadBalance(env)}
	return NewStrategy(defaultGoals)
}
