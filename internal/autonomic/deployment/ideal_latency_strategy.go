package deployment

import (
	"net/http"
	"strconv"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	internalUtils "github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	public "github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"

	log "github.com/sirupsen/logrus"
)

type idealLatencyStrategy struct {
	*basicStrategy
	redirectingTo *utils.Node
	redirectGoal  int
	redirecting   bool
	lbGoal        *deploymentLoadBalanceGoal
	archClient    archimedes.Client
	deployment    *Deployment
}

func newDefaultIdealLatencyStrategy(deployment *Deployment) *idealLatencyStrategy {
	lbGoal := newLoadBalanceGoal(deployment)

	defaultGoals := []goal{
		newIdealLatencyGoal(deployment),
		lbGoal,
	}

	return &idealLatencyStrategy{
		basicStrategy: newBasicStrategy(public.StrategyIdealLatencyId, defaultGoals),
		archClient:    deployment.archFactory.New(internalUtils.ArchimedesLocalHostPort),
		lbGoal:        lbGoal,
		deployment:    deployment,
	}
}

func (i *idealLatencyStrategy) Optimize() actions.Action {
	var (
		nextDomain             domain
		goalToChooseActionFrom goal
		goalActionArgs         []interface{}
	)

	for _, strategyGoal := range i.goals {
		log.Debugf("optimizing %s", strategyGoal.GetId())
		isAlreadyMax, optRange, actionArgs := strategyGoal.Optimize(nextDomain)
		log.Debugf("%s generated optRange %+v", strategyGoal.GetId(), optRange)
		if isAlreadyMax {
			log.Debugf("%s is already maximized", strategyGoal.GetId())
		} else if goalToChooseActionFrom == nil {
			log.Debugf("%s not maximized", strategyGoal.GetId())
			goalToChooseActionFrom = strategyGoal
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
		assertedAction := action.(*actions.RedirectAction)
		assertedAction.SetErrorRedirectingCallback(i.resetRedirecting)

		log.Debugf("redirecting clients from %s to %s", assertedAction.GetOrigin(), assertedAction.GetTarget())
		if i.redirecting {
			// case where i WAS already redirecting

			redirected, status := i.archClient.GetRedirected(i.deployment.DeploymentId)
			if status != http.StatusOK {
				return nil
			}

			if int(redirected) >= i.redirectGoal {
				targetArchClient := i.deployment.archFactory.New(i.redirectingTo.Addr + ":" + strconv.Itoa(
					archimedes.Port))
				status = targetArchClient.RemoveRedirect(i.deployment.DeploymentId)
				if status != http.StatusOK {
					log.Errorf("got status %d while removing redirections for deployment %s at %s", status,
						i.deployment.DeploymentId, i.redirectingTo)
				}
			}
		} else {
			// case where i was NOT yet redirecting
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

func (i *idealLatencyStrategy) resetRedirecting() {
	i.redirecting = false
}
