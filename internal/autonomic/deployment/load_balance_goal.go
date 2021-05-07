package deployment

import (
	"sort"
	"strconv"

	archimedesHTTPClient "github.com/bruno-anjos/archimedesHTTPClient"
	deployer2 "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	autonomicUtils "github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/environment"
	log "github.com/sirupsen/logrus"
)

const (
	maximumLoad            = 300
	staleCyclesNumToRemove = int((float64(archimedesHTTPClient.ResetToFallbackTimeout) * (3. / 2.)) /
		float64(autonomicUtils.DefaultGoalCycleTimeout))

	loadBalanceGoalID          = "GOAL_LOAD_BALANCE"
	overloadedCyclesToRedirect = int((float64(archimedesHTTPClient.CacheExpiringTime) * (3. / 2.)) /
		float64(autonomicUtils.DefaultGoalCycleTimeout))
)

const (
	lbActionTypeArgIndex = iota
	lbFromIndex
	lbAmountIndex
	lbNumArgs
)

type (
	infoValueType struct {
		Load int
		Node *utils.Node
	}
)

type deploymentLoadBalanceGoal struct {
	deployment       *Deployment
	staleCycles      int
	overloadedCycles int
	deplLogger       *log.Entry
}

func newLoadBalanceGoal(deployment *Deployment) *deploymentLoadBalanceGoal {
	return &deploymentLoadBalanceGoal{
		deployment:       deployment,
		staleCycles:      0,
		overloadedCycles: 0,
		deplLogger:       log.WithFields(log.Fields{"DEPL": deployment.DeploymentID, "GOAL": "LOAD_BALANCE"}),
	}
}

func (l *deploymentLoadBalanceGoal) Optimize(optDomain domain) (isAlreadyMax bool, optRange result,
	actionArgs []interface{}) {
	isAlreadyMax = true
	optRange = optDomain
	actionArgs = nil

	candidateIds, sortingCriteria, ok := l.GenerateDomain(nil)
	if !ok {
		return
	}

	l.deplLogger.Debugf("%s generated domain %+v", loadBalanceGoalID, candidateIds)

	filtered := l.Filter(candidateIds, optDomain)
	l.deplLogger.Debugf("%s filtered result %+v", loadBalanceGoalID, filtered)

	ordered := l.Order(filtered, sortingCriteria)
	l.deplLogger.Debugf("%s ordered result %+v", loadBalanceGoalID, ordered)

	optRange, isAlreadyMax = l.Cutoff(ordered, sortingCriteria)
	l.deplLogger.Debugf("%s cutoff result (%t)%+v", loadBalanceGoalID, isAlreadyMax, optRange)

	if !isAlreadyMax {
		isAlreadyMax, optRange, actionArgs = l.handleNotMaximized(optRange, ordered, sortingCriteria)
	} else if l.deployment.Parent != nil {
		remove := l.checkIfShouldBeRemoved()
		if remove {
			isAlreadyMax = false
			actionArgs = make([]interface{}, lbNumArgs)
			actionArgs[lbActionTypeArgIndex] = actions.RemoveDeploymentID
		}
	}

	return isAlreadyMax, optRange, actionArgs
}

func (l *deploymentLoadBalanceGoal) handleNotMaximized(optRange, ordered result,
	sortingCriteria map[string]interface{}) (isAlreadyMax bool, newOptRange result, actionArgs []interface{}) {
	l.staleCycles = 0
	hasAlternatives := l.checkIfHasAlternatives(sortingCriteria)

	if !hasAlternatives {
		// if it doesn't have alternatives try to get a new node
		actionArgs, newOptRange = l.handleOverload(optRange)
		isAlreadyMax = !(len(optRange) > 0)

		return
	}

	// if it has alternatives redirect clients there
	actionArgs = make([]interface{}, lbNumArgs)
	actionArgs[lbActionTypeArgIndex] = actions.RedirectClientsID
	origin := ordered[len(ordered)-1]
	actionArgs[lbFromIndex] = origin
	actionArgs[lbAmountIndex] = sortingCriteria[origin.ID].(infoValueType).Load / 4 //nolint:gomnd

	var (
		filteredRedirectedTargets result
		archClient                = l.deployment.archFactory.New("")
	)

	for _, node := range optRange {
		addr := node.Addr + ":" + strconv.Itoa(archimedes.Port)
		can, _ := archClient.CanRedirectToYou(addr, l.deployment.DeploymentID, Myself.ID)

		l.deplLogger.Debugf("%s deployment %s to redirect: %t", node, l.deployment.DeploymentID, can)

		if can {
			filteredRedirectedTargets = append(filteredRedirectedTargets, node)
		}
	}

	newOptRange = filteredRedirectedTargets
	l.overloadedCycles = 0

	l.deplLogger.Debugf("resetting overloaded cycles")

	return isAlreadyMax, newOptRange, actionArgs
}

func (l *deploymentLoadBalanceGoal) GenerateDomain(_ interface{}) (domain domain, info map[string]interface{},
	success bool) {
	domain = nil
	info = nil
	success = false

	vicinity := l.deployment.Environment.GetVicinity()

	info = map[string]interface{}{}
	load := l.deployment.GetLoad()

	l.deplLogger.Debugf("i have load: %d", load)

	domain = append(domain, Myself)
	info[Myself.ID] = infoValueType{
		Load: load,
		Node: Myself,
	}

	for nodeID, node := range vicinity {
		_, okS := l.deployment.Suspected.Load(nodeID)
		if okS || (l.deployment.Parent != nil && nodeID == l.deployment.Parent.ID) {
			l.deplLogger.Debugf("ignoring %s", nodeID)

			continue
		}

		domain = append(domain, node)

		load = environment.GetLoad(l.deployment.Environment.DemmonCli, l.deployment.DeploymentID, node)
		info[nodeID] = infoValueType{
			Load: load,
			Node: node,
		}

		l.deplLogger.Debugf("%s has load: %d(%d)", nodeID, info[nodeID].(infoValueType).Load, load)
	}

	success = true

	return domain, info, success
}

func (l *deploymentLoadBalanceGoal) Order(candidates domain, sortingCriteria map[string]interface{}) (ordered result) {
	ordered = candidates
	sort.Slice(ordered, func(i, j int) bool {
		loadI := sortingCriteria[ordered[i].ID].(infoValueType).Load
		loadJ := sortingCriteria[ordered[j].ID].(infoValueType).Load

		return loadI < loadJ
	})

	return
}

func (l *deploymentLoadBalanceGoal) Filter(candidates, domain domain) (filtered result) {
	return defaultFilter(candidates, domain)
}

func (l *deploymentLoadBalanceGoal) Cutoff(candidates domain, candidatesCriteria map[string]interface{}) (cutoff result,
	maxed bool) {
	cutoff = nil
	maxed = true

	myLoad := candidatesCriteria[Myself.ID].(infoValueType).Load

	if myLoad > maximumLoad {
		l.deplLogger.Debugf("im overloaded: %d (%d)", myLoad, overloadedCyclesToRedirect)

		if l.overloadedCycles == overloadedCyclesToRedirect {
			maxed = false
		}

		l.overloadedCycles++
	} else {
		l.deplLogger.Debugf("resetting overloaded cycles")
		l.overloadedCycles = 0
	}

	for _, candidate := range candidates {
		if candidatesCriteria[candidate.ID].(infoValueType).Load <= maximumLoad {
			cutoff = append(cutoff, candidate)
		}
	}

	if len(cutoff) == 0 {
		maxed = true
	}

	return cutoff, maxed
}

func (l *deploymentLoadBalanceGoal) GenerateAction(targets []*utils.Node, args ...interface{}) actions.Action {
	l.deplLogger.Debugf("generating action %s", (args[lbActionTypeArgIndex]).(string))

	switch args[lbActionTypeArgIndex].(string) {
	case actions.ExtendDeploymentID:
		location := environment.GetLocation(l.deployment.Environment.DemmonCli, targets[0])

		toExclude := map[string]interface{}{}

		l.deployment.Blacklist.Range(func(key, value interface{}) bool {
			nodeID := key.(string)
			toExclude[nodeID] = nil

			return true
		})

		l.deployment.Exploring.Range(func(key, value interface{}) bool {
			nodeID := key.(string)
			toExclude[nodeID] = nil

			return true
		})

		return actions.NewExtendDeploymentAction(l.deployment.DeploymentID, targets[0], deployer2.NotExploringTTL,
			nil, location, toExclude, l.deployment.setNodeAsExploring, l.deployment.deplFactory)
	case actions.RedirectClientsID:
		return actions.NewRedirectAction(l.deployment.DeploymentID, args[lbFromIndex].(*utils.Node), targets[0],
			args[lbAmountIndex].(int))
	case actions.RemoveDeploymentID:
		return actions.NewRemoveDeploymentAction(l.deployment.DeploymentID)
	}

	return nil
}

func (l *deploymentLoadBalanceGoal) GetID() string {
	return loadBalanceGoalID
}

func (l *deploymentLoadBalanceGoal) handleOverload(candidates result) (
	actionArgs []interface{}, newOptRange result) {
	actionArgs = make([]interface{}, lbNumArgs)

	actionArgs[lbActionTypeArgIndex] = actions.ExtendDeploymentID
	deplClient := l.deployment.deplFactory.New()

	for _, candidate := range candidates {
		_, okC := l.deployment.Children.Load(candidate)
		if !okC {
			addr := candidate.Addr + ":" + strconv.Itoa(deployer.Port)

			hasDeployment, _ := deplClient.HasDeployment(addr, l.deployment.DeploymentID)
			if hasDeployment {
				continue
			}

			newOptRange = append(newOptRange, candidate)

			break
		}
	}

	l.deplLogger.Debugf("%s new opt range %+v", loadBalanceGoalID, newOptRange)

	return
}

func (l *deploymentLoadBalanceGoal) checkIfShouldBeRemoved() bool {
	hasChildren := false

	l.deployment.Children.Range(func(key, value interface{}) bool {
		hasChildren = true

		return false
	})

	if hasChildren {
		l.staleCycles = 0
		l.deplLogger.Debugf("%s should NOT be removed, because it has children", l.deployment.DeploymentID)

		return false
	}

	load := l.deployment.GetLoad()
	if load > 0 {
		l.staleCycles = 0
		l.deplLogger.Debugf("%s should NOT be removed, because it has load %d", l.deployment.DeploymentID, load)

		return false
	}

	l.staleCycles++
	l.deplLogger.Debugf("increased stale cycles to %d(%d)", l.staleCycles, staleCyclesNumToRemove)

	return l.staleCycles == staleCyclesNumToRemove
}

func (l *deploymentLoadBalanceGoal) checkIfHasAlternatives(sortingCriteria map[string]interface{}) (
	hasAlternatives bool) {
	myLoad := sortingCriteria[Myself.ID].(infoValueType).Load

	deplClient := l.deployment.deplFactory.New()

	for _, value := range sortingCriteria {
		infoValue := value.(infoValueType)

		load := infoValue.Load
		if float64(load) < maximumLoad && float64(load) < float64(myLoad)/2. {
			addr := infoValue.Node.Addr + ":" + strconv.Itoa(deployer.Port)

			hasDeployment, _ := deplClient.HasDeployment(addr, l.deployment.DeploymentID)
			if hasDeployment {
				hasAlternatives = true

				break
			}
		}
	}

	return
}

func (l *deploymentLoadBalanceGoal) errorRedirecting() {
}
