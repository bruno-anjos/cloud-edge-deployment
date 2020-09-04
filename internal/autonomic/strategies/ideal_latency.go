package strategies

import (
	"net/http"
	"sync"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/goals"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/goals/service_goals"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
)

const (
	StrategyIdealLatencyId = "STRATEGY_IDEAL_LATENCY"
)

type idealLatencyStrategy struct {
	*BasicStrategy
	redirected          int
	redirecting         bool
	redirectingTo       string
	redirectInitialLoad float64
	lbGoal              *service_goals.LoadBalance
	archClient          *archimedes.Client
	env                 *autonomic.Environment
	serviceId           string
}

func NewDefaultIdealLatencyStrategy(serviceId string, serviceChildren *sync.Map,
	env *autonomic.Environment) *idealLatencyStrategy {
	lbGoal := service_goals.NewLoadBalance(serviceId, env)

	defaultGoals := []goals.Goal{
		service_goals.NewIdealLatency(serviceId, serviceChildren, env),
		lbGoal,
	}

	return &idealLatencyStrategy{
		BasicStrategy: NewBasicStrategy(defaultGoals),
		redirected:    0,
		archClient:    archimedes.NewArchimedesClient(archimedes.ArchimedesServiceName),
		serviceId:     serviceId,
		env:           env,
		lbGoal:        lbGoal,
	}
}

func (i *idealLatencyStrategy) Optimize() actions.Action {
	var (
		nextDomain             goals.Domain
		goalToChooseActionFrom goals.Goal
		goalActionArgs         []interface{}
	)

	for _, goal := range i.goals {
		isAlreadyMax, optRange, actionArgs := goal.Optimize(nextDomain)
		if isAlreadyMax {
			nextDomain = optRange
		} else {
			goalToChooseActionFrom = goal
			goalActionArgs = actionArgs
		}
	}

	if goalToChooseActionFrom == nil || nextDomain == nil {
		return nil
	}

	action := goalToChooseActionFrom.GenerateAction(nextDomain[0], goalActionArgs)
	if action.GetActionId() == actions.REDIRECT_CLIENTS_ID {
		if i.redirecting {
			// case where i WAS already redirecting

			redirected, status := i.archClient.GetRedirected(i.serviceId)
			if status != http.StatusOK {
				return nil
			}
			i.redirected = int(redirected)

			value, _ := i.env.GetMetric(autonomic.METRIC_LOAD_PER_SERVICE_IN_CHILD)
			currLoad := value.(float64)
			loadDiff := currLoad - i.redirectInitialLoad

			// TODO this is not that smart
			if loadDiff > 0.75 {
				i.lbGoal.DecreaseMigrationGroupSize()
			} else if loadDiff < 0.25 {
				i.lbGoal.IncreaseMigrationGroupSize()
			}
		} else {
			// case where i was NOT yet redirecting

			i.redirectingTo = nextDomain[0]
			i.redirected = 0
			i.redirecting = true
			value, _ := i.env.GetMetric(autonomic.METRIC_LOAD_PER_SERVICE_IN_CHILD)
			i.redirectInitialLoad = value.(float64)
			i.lbGoal.ResetMigrationGroupSize()
		}
	} else if i.redirecting {
		i.redirecting = false
		i.archClient.RemoveRedirect(i.serviceId)
	}

	return action
}
