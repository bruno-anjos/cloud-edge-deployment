package deployment

import (
	"net/http"
	"strconv"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/servers"
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
		basicStrategy: newBasicStrategy(public.StrategyIdealLatencyID, defaultGoals),
		archClient:    deployment.archFactory.New(servers.ArchimedesLocalHostPort),
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
		log.Debugf("optimizing %s", strategyGoal.GetID())

		isAlreadyMax, optRange, actionArgs := strategyGoal.Optimize(nextDomain)
		log.Debugf("%s generated optRange %+v", strategyGoal.GetID(), optRange)

		if isAlreadyMax {
			log.Debugf("%s is already maximized", strategyGoal.GetID())
		} else if goalToChooseActionFrom == nil {
			log.Debugf("%s not maximized", strategyGoal.GetID())
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

	log.Debugf("generated action of type %s", action.GetActionID())

	success := true
	if action.GetActionID() == actions.RedirectClientsID {
		success = i.handleRedirectAction(action)
	} else if i.redirecting {
		i.redirecting = false
		i.archClient.RemoveRedirect(i.deployment.DeploymentID)
	}

	if !success {
		return nil
	}

	return action
}

func (i *idealLatencyStrategy) handleRedirectAction(action actions.Action) (success bool) {
	success = true
	assertedAction := action.(*actions.RedirectAction)
	assertedAction.SetErrorRedirectingCallback(i.resetRedirecting)

	log.Debugf("redirecting clients from %s to %s", assertedAction.GetOrigin(), assertedAction.GetTarget())

	if i.redirecting {
		success = i.handleRedirecting()
	} else {
		// case where i was NOT yet redirecting
		i.redirecting = true
		i.redirectGoal = assertedAction.GetAmount()
		i.redirectingTo = assertedAction.GetTarget()
	}

	return success
}

func (i *idealLatencyStrategy) handleRedirecting() bool {
	// case where i WAS already redirecting
	redirected, status := i.archClient.GetRedirected(i.deployment.DeploymentID)
	if status != http.StatusOK {
		return false
	}

	if int(redirected) >= i.redirectGoal {
		targetArchClient := i.deployment.archFactory.New(i.redirectingTo.Addr + ":" + strconv.Itoa(
			archimedes.Port))

		status = targetArchClient.RemoveRedirect(i.deployment.DeploymentID)
		if status != http.StatusOK {
			log.Errorf("got status %d while removing redirections for deployment %s at %s", status,
				i.deployment.DeploymentID, i.redirectingTo)
		}
	}

	return true
}

func (i *idealLatencyStrategy) resetRedirecting() {
	i.redirecting = false
}
