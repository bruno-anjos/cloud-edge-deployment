package service

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/metrics"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	publicUtils "github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
)

const (
	nodeLoadBalanceId = "GOAL_NODE_LOAD_BALANCE"
)

const (
	nlbActionTypeArgIndex = iota
	nlbParentIdx
	nlbChildrenIdx
	nlbNumArgs
)

type (
	nodeCriteria struct {
		Deployments []string
	}
)

type nodeLoadBalanceGoal struct {
	service      *Service
	dependencies []string
}

func newNodeLoadBalanceGoal(service *Service) *nodeLoadBalanceGoal {
	return &nodeLoadBalanceGoal{
		service: service,
	}
}

func (nl *nodeLoadBalanceGoal) Optimize(optDomain Domain) (isAlreadyMax bool, optRange Range, actionArgs []interface{}) {
	isAlreadyMax = true
	optRange = nil

	candidateIds, criteria, ok := nl.GenerateDomain(nil)
	if !ok {
		return
	}

	filtered := nl.Filter(candidateIds, optDomain)
	log.Debugf("%s filtered result %+v", loadBalanceGoalId, filtered)

	ordered := nl.Order(filtered, criteria)
	log.Debugf("%s ordered result %+v", loadBalanceGoalId, ordered)

	optRange, isAlreadyMax = nl.Cutoff(ordered, criteria)
	log.Debugf("%s cutoff result (%t)%+v", loadBalanceGoalId, isAlreadyMax, optRange)

	if !isAlreadyMax {
	}

	return
}

func (nl *nodeLoadBalanceGoal) GenerateDomain(_ interface{}) (domain Domain, info map[string]interface{},
	success bool) {
	domain = nil

	info = nil
	success = false

	value, ok := nl.service.Environment.GetMetric(metrics.MetricLocationInVicinity)
	if !ok {
		log.Debugf("no value for metric %s", metrics.MetricLocationInVicinity)
		return nil, nil, false
	}

	var locationsInVicinity map[string]publicUtils.Location
	err := mapstructure.Decode(value, &locationsInVicinity)
	if err != nil {
		panic(err)
	}

	value, ok = nl.service.Environment.GetMetric(metrics.MetricNodeAddr)
	if !ok {
		log.Debugf("no value for metric %s", metrics.MetricNodeAddr)
		return nil, nil, false
	}

	myself := value.(string)
	info = map[string]interface{}{}
	deplClient := deployer.NewDeployerClient("")

	for nodeId := range locationsInVicinity {
		_, okS := nl.service.Suspected.Load(nodeId)
		if okS || nodeId == myself || nodeId == nl.service.ParentId {
			log.Debugf("ignoring %s", nodeId)
			continue
		}

		domain = append(domain, nodeId)

		deplClient.SetHostPort(nodeId + ":" + strconv.Itoa(deployer.Port))
		deployments, status := deplClient.GetDeployments()
		if status != http.StatusOK {
			info[nodeId] = &nodeCriteria{Deployments: []string{}}
		} else {
			info[nodeId] = &nodeCriteria{Deployments: deployments}
		}

		log.Debugf("%s has deployments: %v", nodeId, deployments)
	}

	success = true

	return
}

func (nl *nodeLoadBalanceGoal) Filter(candidates, domain Domain) (filtered Range) {
	return DefaultFilter(candidates, domain)
}

func (nl *nodeLoadBalanceGoal) Order(candidates Domain, sortingCriteria map[string]interface{}) (ordered Range) {
	ordered = candidates
	sort.Slice(ordered, func(i, j int) bool {
		loadI := len(sortingCriteria[ordered[i]].(*nodeCriteria).Deployments)
		loadJ := len(sortingCriteria[ordered[j]].(*nodeCriteria).Deployments)
		return loadI < loadJ
	})

	return
}

func (nl *nodeLoadBalanceGoal) Cutoff(candidates Domain, candidatesCriteria map[string]interface{}) (cutoff Range,
	maxed bool) {

	cutoff = nil
	maxed = true

	if len(candidates) < 2 {
		cutoff = candidates
		return
	}

	leastBusy := candidates[0]
	leastAmountOfServices := len(candidatesCriteria[leastBusy].(*nodeCriteria).Deployments)

	for _, candidate := range candidates {
		if len(candidatesCriteria[candidate].(*nodeCriteria).Deployments) < (leastAmountOfServices+1)*2 {
			cutoff = append(cutoff, candidate)
		} else {
			maxed = false
		}
	}

	return
}

func (nl *nodeLoadBalanceGoal) GenerateAction(target string, args ...interface{}) actions.Action {
	return nil
}

func (nl *nodeLoadBalanceGoal) GetId() string {
	return nodeLoadBalanceId
}

func (nl *nodeLoadBalanceGoal) TestDryRun() bool {
	return true
}

func (nl *nodeLoadBalanceGoal) GetDependencies() (metrics []string) {
	return nil
}