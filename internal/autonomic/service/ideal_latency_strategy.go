package service

import (
	"net/http"
	"strconv"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/metrics"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	public "github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	log "github.com/sirupsen/logrus"
)

type idealLatencyStrategy struct {
	*basicStrategy
	redirected          int
	redirecting         bool
	redirectingTo       string
	redirectInitialLoad float64
	lbGoal              *loadBalanceGoal
	archClient          *archimedes.Client
	service             *Service
}

func newDefaultIdealLatencyStrategy(service *Service) *idealLatencyStrategy {
	lbGoal := newLoadBalanceGoal(service)

	defaultGoals := []Goal{
		NewIdealLatencyGoal(service),
		lbGoal,
	}

	return &idealLatencyStrategy{
		basicStrategy: newBasicStrategy(public.StrategyIdealLatencyId, defaultGoals),
		redirected:    0,
		archClient:    archimedes.NewArchimedesClient(archimedes.DefaultHostPort),
		lbGoal:        lbGoal,
		service:       service,
	}
}

func (i *idealLatencyStrategy) Optimize() actions.Action {
	var (
		nextDomain             Domain
		goalToChooseActionFrom Goal
		goalActionArgs         []interface{}
	)

	for _, goal := range i.goals {
		log.Debugf("optimizing %s", goal.GetId())
		isAlreadyMax, optRange, actionArgs := goal.Optimize(nextDomain)
		log.Debugf("%s generated optRange %+v", goal.GetId(), optRange)
		if isAlreadyMax {
			log.Debugf("%s is already maximized", goal.GetId())
		} else if goalToChooseActionFrom == nil {
			log.Debugf("%s not maximized", goal.GetId())
			goalToChooseActionFrom = goal
			goalActionArgs = actionArgs
		}

		if optRange != nil {
			nextDomain = optRange
		}
	}

	if goalToChooseActionFrom == nil || nextDomain == nil || len(nextDomain) == 0 {
		return nil
	}

	action := goalToChooseActionFrom.GenerateAction(nextDomain[0], goalActionArgs...)
	log.Debugf("generated action of type %s", action.GetActionId())
	if action.GetActionId() == actions.RedirectClientsId {
		if i.redirecting {
			// case where i WAS already redirecting

			redirected, status := i.archClient.GetRedirected(i.service.ServiceId)
			if status != http.StatusOK {
				return nil
			}
			i.redirected = int(redirected)

			loadPerServiceChild := metrics.GetLoadPerServiceInChildMetricId(i.service.ServiceId, i.redirectingTo)
			value, _ := i.service.Environment.GetMetric(loadPerServiceChild)
			currLoad := value.(float64)
			loadDiff := currLoad - i.redirectInitialLoad

			// TODO this is not that smart
			if loadDiff > 0.75 {
				i.lbGoal.decreaseMigrationGroupSize()
			} else if loadDiff < 0.25 {
				i.lbGoal.increaseMigrationGroupSize()
			}
		} else {
			// case where i was NOT yet redirecting

			i.redirectingTo = nextDomain[0]
			i.redirected = 0
			i.redirecting = true
			archClient := archimedes.NewArchimedesClient(i.redirectingTo + ":" + strconv.Itoa(archimedes.Port))
			load, _ := archClient.GetLoad(i.service.ServiceId)
			i.redirectInitialLoad = load
			i.lbGoal.resetMigrationGroupSize()
		}
	} else if i.redirecting {
		i.redirecting = false
		i.archClient.RemoveRedirect(i.service.ServiceId)
	}

	return action
}
