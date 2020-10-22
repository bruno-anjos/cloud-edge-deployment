package service

import (
	"net/http"
	"strconv"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	public "github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	log "github.com/sirupsen/logrus"
)

type idealLatencyStrategy struct {
	*basicStrategy
	redirectingTo string
	redirectGoal  int
	redirecting   bool
	lbGoal        *serviceLoadBalanceGoal
	archClient    *archimedes.Client
	service       *Service
}

func newDefaultIdealLatencyStrategy(service *Service) *idealLatencyStrategy {
	lbGoal := newLoadBalanceGoal(service)

	defaultGoals := []Goal{
		newIdealLatencyGoal(service),
		lbGoal,
	}

	return &idealLatencyStrategy{
		basicStrategy: newBasicStrategy(public.StrategyIdealLatencyId, defaultGoals),
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

			if int(redirected) >= i.redirectGoal {
				targetArchClient := archimedes.NewArchimedesClient(i.redirectingTo + ":" + strconv.Itoa(archimedes.Port))
				status = targetArchClient.RemoveRedirect(i.service.ServiceId)
				if status != http.StatusOK {
					log.Errorf("got status %d while removing redirections for service %s at %s", status,
						i.service.ServiceId, i.redirectingTo)
				}
			}

		} else {
			// case where i was NOT yet redirecting
			assertedAction := action.(*actions.RedirectAction)
			i.redirecting = true
			i.redirectGoal = assertedAction.GetAmount()
			i.redirectingTo = assertedAction.GetTarget()
		}
	} else if i.redirecting {
		i.redirecting = false
		i.archClient.RemoveRedirect(i.service.ServiceId)
	}

	return action
}
