package deployment

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
	lbGoal        *deploymentLoadBalanceGoal
	archClient    *archimedes.Client
	deployment    *Deployment
}

func newDefaultIdealLatencyStrategy(deployment *Deployment) *idealLatencyStrategy {
	lbGoal := newLoadBalanceGoal(deployment)

	defaultGoals := []Goal{
		newIdealLatencyGoal(deployment),
		lbGoal,
	}

	return &idealLatencyStrategy{
		basicStrategy: newBasicStrategy(public.StrategyIdealLatencyId, defaultGoals),
		archClient:    archimedes.NewArchimedesClient(archimedes.DefaultHostPort),
		lbGoal:        lbGoal,
		deployment:    deployment,
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

	action := goalToChooseActionFrom.GenerateAction(nextDomain, goalActionArgs...)
	if action == nil {
		return action
	}

	log.Debugf("generated action of type %s", action.GetActionId())
	if action.GetActionId() == actions.RedirectClientsId {
		if i.redirecting {
			// case where i WAS already redirecting

			redirected, status := i.archClient.GetRedirected(i.deployment.DeploymentId)
			if status != http.StatusOK {
				return nil
			}

			if int(redirected) >= i.redirectGoal {
				targetArchClient := archimedes.NewArchimedesClient(i.redirectingTo + ":" + strconv.Itoa(archimedes.Port))
				status = targetArchClient.RemoveRedirect(i.deployment.DeploymentId)
				if status != http.StatusOK {
					log.Errorf("got status %d while removing redirections for deployment %s at %s", status,
						i.deployment.DeploymentId, i.redirectingTo)
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
		i.archClient.RemoveRedirect(i.deployment.DeploymentId)
	}

	return action
}
