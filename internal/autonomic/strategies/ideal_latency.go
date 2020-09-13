package strategies

import (
	"net/http"
	"sync"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/environment"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/goals"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/goals/service_goals"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/metrics"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	log "github.com/sirupsen/logrus"
)

const (
	StrategyIdealLatencyId = "STRATEGY_IDEAL_LATENCY"
)

type idealLatencyStrategy struct {
	*basicStrategy
	redirected          int
	redirecting         bool
	redirectingTo       string
	redirectInitialLoad float64
	lbGoal              *service_goals.LoadBalance
	archClient          *archimedes.Client
	env                 *environment.Environment
	serviceId           string
}

func NewDefaultIdealLatencyStrategy(serviceId string, serviceChildren, suspected *sync.Map, parentId **string,
	env *environment.Environment) *idealLatencyStrategy {
	lbGoal := service_goals.NewLoadBalance(serviceId, serviceChildren, suspected, parentId, env)

	defaultGoals := []goals.Goal{
		service_goals.NewIdealLatency(serviceId, serviceChildren, suspected, parentId, env),
		lbGoal,
	}

	return &idealLatencyStrategy{
		basicStrategy: newBasicStrategy(StrategyIdealLatencyId, defaultGoals),
		redirected:    0,
		archClient:    archimedes.NewArchimedesClient(archimedes.DefaultHostPort),
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
		log.Debugf("optimizing %s", goal.GetId())
		isAlreadyMax, optRange, actionArgs := goal.Optimize(nextDomain)
		log.Debugf("%s generated optRange %+v", goal.GetId(), optRange)
		if isAlreadyMax {
			log.Debugf("%s is already maximized", goal.GetId())
		} else {
			log.Debugf("%s not maximized", goal.GetId())
			goalToChooseActionFrom = goal
			goalActionArgs = actionArgs
		}

		if optRange != nil {
			nextDomain = optRange
		}
	}

	if goalToChooseActionFrom == nil || nextDomain == nil {
		return nil
	}

	action := goalToChooseActionFrom.GenerateAction(nextDomain[0], goalActionArgs...)
	log.Debugf("generated action of type %s", action.GetActionId())
	if action.GetActionId() == actions.REDIRECT_CLIENTS_ID {
		if i.redirecting {
			// case where i WAS already redirecting

			redirected, status := i.archClient.GetRedirected(i.serviceId)
			if status != http.StatusOK {
				return nil
			}
			i.redirected = int(redirected)

			loadPerServiceChild := metrics.GetLoadPerServiceInChildMetricId(i.serviceId, i.redirectingTo)
			value, _ := i.env.GetMetric(loadPerServiceChild)
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
			loadPerServiceChild := metrics.GetLoadPerServiceInChildMetricId(i.serviceId, i.redirectingTo)
			value, _ := i.env.GetMetric(loadPerServiceChild)
			i.redirectInitialLoad = value.(float64)
			i.lbGoal.ResetMigrationGroupSize()
		}
	} else if i.redirecting {
		i.redirecting = false
		i.archClient.RemoveRedirect(i.serviceId)
	}

	return action
}
