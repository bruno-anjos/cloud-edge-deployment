package deployment

import (
	"net/http"
	"sort"
	"strconv"

	archimedesHTTPClient "github.com/bruno-anjos/archimedesHTTPClient"
	deployer2 "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/metrics"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/utils"
	utils2 "github.com/bruno-anjos/cloud-edge-deployment/internal/utils"


	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
)

const (
	maximumLoad            = 300
	staleCyclesNumToRemove = int((float64(archimedesHTTPClient.ResetToFallbackTimeout) * (3. / 2.)) /
		float64(utils.DefaultGoalCycleTimeout))

	loadBalanceGoalId          = "GOAL_LOAD_BALANCE"
	overloadedCyclesToRedirect = int((float64(archimedesHTTPClient.CacheExpiringTime) * (3. / 2.)) /
		float64(utils.DefaultGoalCycleTimeout))
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
		Node *utils2.Node
	}
)

type deploymentLoadBalanceGoal struct {
	deployment       *Deployment
	dependencies     []string
	staleCycles      int
	overloadedCycles int
}

func newLoadBalanceGoal(deployment *Deployment) *deploymentLoadBalanceGoal {
	dependencies := []string{
		metrics.GetAggLoadPerDeploymentInChildrenMetricId(deployment.DeploymentId),
		metrics.GetLoadPerDeploymentInChildrenMetricId(deployment.DeploymentId),
	}

	return &deploymentLoadBalanceGoal{
		deployment:   deployment,
		dependencies: dependencies,
	}
}

func (l *deploymentLoadBalanceGoal) Optimize(optDomain Domain) (isAlreadyMax bool, optRange Range,
	actionArgs []interface{}) {
	isAlreadyMax = true
	optRange = optDomain
	actionArgs = nil

	candidateIds, sortingCriteria, ok := l.GenerateDomain(nil)
	if !ok {
		return
	}
	log.Debugf("%s generated domain %+v", loadBalanceGoalId, candidateIds)

	filtered := l.Filter(candidateIds, optDomain)
	log.Debugf("%s filtered result %+v", loadBalanceGoalId, filtered)

	ordered := l.Order(filtered, sortingCriteria)
	log.Debugf("%s ordered result %+v", loadBalanceGoalId, ordered)

	optRange, isAlreadyMax = l.Cutoff(ordered, sortingCriteria)
	log.Debugf("%s cutoff result (%t)%+v", loadBalanceGoalId, isAlreadyMax, optRange)

	if !isAlreadyMax {
		l.staleCycles = 0
		hasAlternatives := l.checkIfHasAlternatives(sortingCriteria)

		if !hasAlternatives {
			// if it doesn't have alternatives try to get a new node
			actionArgs, optRange = l.handleOverload(optRange)
			isAlreadyMax = !(len(optRange) > 0)
		} else {
			// if it has alternatives redirect clients there
			actionArgs = make([]interface{}, lbNumArgs)
			actionArgs[lbActionTypeArgIndex] = actions.RedirectClientsId
			origin := ordered[len(ordered)-1]
			actionArgs[lbFromIndex] = origin
			actionArgs[lbAmountIndex] = sortingCriteria[origin.Id].(infoValueType).Load / 4

			var filteredRedirectedTargets Range
			archClient := archimedes.NewArchimedesClient("")
			for _, node := range optRange {
				archClient.SetHostPort(node.Addr + ":" + strconv.Itoa(utils2.ArchimedesPort))
				can, _ := archClient.CanRedirectToYou(l.deployment.DeploymentId, Myself.Id)
				log.Debugf("%s deployment %s to redirect: %t", node, l.deployment.DeploymentId, can)
				if can {
					filteredRedirectedTargets = append(filteredRedirectedTargets, node)
				}
			}
			optRange = filteredRedirectedTargets
			l.overloadedCycles = 0
			log.Debugf("resetting overloaded cycles")
		}
	} else if l.deployment.Parent != nil {
		remove := l.checkIfShouldBeRemoved()
		if remove {
			isAlreadyMax = false
			actionArgs = make([]interface{}, lbNumArgs)
			actionArgs[lbActionTypeArgIndex] = actions.RemoveDeploymentId
		}
	}

	return
}

func (l *deploymentLoadBalanceGoal) GenerateDomain(_ interface{}) (domain Domain, info map[string]interface{},
	success bool) {
	domain = nil
	info = nil
	success = false

	value, ok := l.deployment.Environment.GetMetric(metrics.MetricLocationInVicinity)
	if !ok {
		log.Debugf("no value for metric %s", metrics.MetricLocationInVicinity)
		return nil, nil, false
	}

	var locationsInVicinity metrics.VicinityMetric
	err := mapstructure.Decode(value, &locationsInVicinity)
	if err != nil {
		panic(err)
	}

	info = map[string]interface{}{}
	archClient := archimedes.NewArchimedesClient(Myself.Addr + ":" + strconv.Itoa(utils2.ArchimedesPort))
	load, status := archClient.GetLoad(l.deployment.DeploymentId)
	if status != http.StatusOK {
		load = 0
	}

	domain = append(domain, Myself)
	info[Myself.Id] = infoValueType{
		Load: load,
		Node: Myself,
	}

	for nodeId, node := range locationsInVicinity.Nodes {
		_, okS := l.deployment.Suspected.Load(nodeId)
		if okS || (l.deployment.Parent != nil && nodeId == l.deployment.Parent.Id) {
			log.Debugf("ignoring %s", nodeId)
			continue
		}
		domain = append(domain, node)
		archClient.SetHostPort(node.Addr + ":" + strconv.Itoa(utils2.ArchimedesPort))
		load, status = archClient.GetLoad(l.deployment.DeploymentId)
		if status != http.StatusOK {
			info[nodeId] = infoValueType{
				Load: 0,
				Node: node,
			}
		} else {
			info[nodeId] = infoValueType{
				Load: load,
				Node: node,
			}
		}
		log.Debugf("%s has load: %d(%d)", nodeId, info[nodeId].(infoValueType).Load, load)
	}

	success = true

	return
}

func (l *deploymentLoadBalanceGoal) Order(candidates Domain, sortingCriteria map[string]interface{}) (ordered Range) {
	ordered = candidates
	sort.Slice(ordered, func(i, j int) bool {
		loadI := sortingCriteria[ordered[i].Id].(infoValueType).Load
		loadJ := sortingCriteria[ordered[j].Id].(infoValueType).Load
		return loadI < loadJ
	})

	return
}

func (l *deploymentLoadBalanceGoal) Filter(candidates, domain Domain) (filtered Range) {
	return DefaultFilter(candidates, domain)
}

func (l *deploymentLoadBalanceGoal) Cutoff(candidates Domain, candidatesCriteria map[string]interface{}) (cutoff Range,
	maxed bool) {

	cutoff = nil
	maxed = true

	myLoad := candidatesCriteria[Myself.Id].(infoValueType).Load

	if myLoad > maximumLoad {
		log.Debugf("im overloaded: %d (%d)", myLoad, overloadedCyclesToRedirect)
		if l.overloadedCycles == overloadedCyclesToRedirect {
			maxed = false
		}
		l.overloadedCycles++
	} else {
		log.Debugf("resetting overloaded cycles")
		l.overloadedCycles = 0
	}

	for _, candidate := range candidates {
		if candidatesCriteria[candidate.Id].(infoValueType).Load <= maximumLoad {
			cutoff = append(cutoff, candidate)
		}
	}

	if len(cutoff) == 0 {
		maxed = true
	}

	return
}

func (l *deploymentLoadBalanceGoal) GenerateAction(targets []*utils2.Node, args ...interface{}) actions.Action {
	log.Debugf("generating action %s", (args[lbActionTypeArgIndex]).(string))

	switch args[lbActionTypeArgIndex].(string) {
	case actions.ExtendDeploymentId:
		autoClient := client.NewAutonomicClient(targets[0].Addr + ":" + strconv.Itoa(utils2.AutonomicPort))
		location, status := autoClient.GetLocation()
		if status != http.StatusOK {
			log.Errorf("got status %d while getting %s location", status, targets[0])
			return nil
		}

		toExclude := map[string]interface{}{}
		l.deployment.Blacklist.Range(func(key, value interface{}) bool {
			nodeId := key.(string)
			toExclude[nodeId] = nil
			return true
		})
		l.deployment.Exploring.Range(func(key, value interface{}) bool {
			nodeId := key.(string)
			toExclude[nodeId] = nil
			return true
		})

		return actions.NewExtendDeploymentAction(l.deployment.DeploymentId, targets[0], deployer2.NotExploringTTL,
			nil, location, toExclude, l.deployment.setNodeAsExploring)
	case actions.RedirectClientsId:
		return actions.NewRedirectAction(l.deployment.DeploymentId, args[lbFromIndex].(*utils2.Node), targets[0],
			args[lbAmountIndex].(int))
	case actions.RemoveDeploymentId:
		return actions.NewRemoveDeploymentAction(l.deployment.DeploymentId)
	}

	return nil
}

func (l *deploymentLoadBalanceGoal) GetDependencies() (metrics []string) {
	return l.dependencies
}

func (l *deploymentLoadBalanceGoal) GetId() string {
	return loadBalanceGoalId
}

func (l *deploymentLoadBalanceGoal) handleOverload(candidates Range) (
	actionArgs []interface{}, newOptRange Range) {
	actionArgs = make([]interface{}, lbNumArgs, lbNumArgs)

	actionArgs[lbActionTypeArgIndex] = actions.ExtendDeploymentId
	deplClient := client2.NewDeployerClient("")
	for _, candidate := range candidates {
		_, okC := l.deployment.Children.Load(candidate)
		if !okC {
			deplClient.SetHostPort(candidate.Addr + ":" + strconv.Itoa(utils2.DeployerPort))
			hasDeployment, _ := deplClient.HasDeployment(l.deployment.DeploymentId)
			if hasDeployment {
				continue
			}
			newOptRange = append(newOptRange, candidate)
			break
		}
	}

	log.Debugf("%s new opt range %+v", loadBalanceGoalId, newOptRange)

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
		log.Debugf("%s should NOT be removed, because it has children", l.deployment.DeploymentId)
		return false
	}

	archClient := archimedes.NewArchimedesClient(utils2.ArchimedesLocalHostPort)
	load, status := archClient.GetLoad(l.deployment.DeploymentId)
	if status != http.StatusOK {
		log.Errorf("got status %d when asking for load for deployment %s", status, l.deployment.DeploymentId)
		return false
	}

	if load > 0 {
		l.staleCycles = 0
		log.Debugf("%s should NOT be removed, because it has load %d", l.deployment.DeploymentId, load)
		return false
	}

	l.staleCycles++
	log.Debugf("increased stale cycles to %d(%d)", l.staleCycles, staleCyclesNumToRemove)

	return l.staleCycles == staleCyclesNumToRemove
}

func (l *deploymentLoadBalanceGoal) checkIfHasAlternatives(sortingCriteria map[string]interface{}) (hasAlternatives bool) {
	myLoad := sortingCriteria[Myself.Id].(infoValueType).Load

	var (
		deplClient = client2.NewDeployerClient("")
	)
	for _, value := range sortingCriteria {
		infoValue := value.(infoValueType)
		load := infoValue.Load
		if float64(load) < maximumLoad && float64(load) < float64(myLoad)/2. {
			deplClient.SetHostPort(infoValue.Node.Addr + ":" + strconv.Itoa(utils2.DeployerPort))
			hasDeployment, _ := deplClient.HasDeployment(l.deployment.DeploymentId)
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
