package service_goals

import (
	"math"
	"sort"
	"sync"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/environment"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/goals"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/metrics"
	log "github.com/sirupsen/logrus"
)

const (
	processingThreshold = 0.8

	maximumDistancePercentage   = 1.1
	equivalenDistancePercentage = 0.8
	loadToAddServiceThreshold   = 0.7

	ilArgsNum = 2

	idealLatencyGoalId = "GOAL_IDEAL_LATENCY"
)

const (
	ilActionTypeArgIndex = iota
	ilFromIndex
)

type NodeWithLocation struct {
	NodeId   string
	Location float64
}

type (
	serviceChildrenMapKey   = string
	serviceChildrenMapValue = *NodeWithLocation
)

type nodeWithDistance struct {
	NodeId             string
	DistancePercentage float64
}

type idealLatency struct {
	serviceId       string
	serviceChildren *sync.Map
	suspected       *sync.Map
	environment     *environment.Environment
	dependencies    []string
	parentId        **string
}

func NewIdealLatency(serviceId string, serviceChildren, suspected *sync.Map,
	parentId **string, env *environment.Environment) *idealLatency {
	dependencies := []string{
		metrics.GetProcessingTimePerServiceMetricId(serviceId),
		metrics.GetClientLatencyPerServiceMetricId(serviceId),
		metrics.MetricLocation,
		metrics.GetAverageClientLocationPerServiceMetricId(serviceId),
		metrics.MetricLocationInVicinity,
		metrics.GetNumInstancesMetricId(serviceId),
	}

	goal := &idealLatency{
		serviceId:       serviceId,
		serviceChildren: serviceChildren,
		suspected:       suspected,
		environment:     env,
		dependencies:    dependencies,
		parentId:        parentId,
	}

	return goal
}

func (i *idealLatency) Optimize(optDomain goals.Domain) (isAlreadyMax bool, optRange goals.Range,
	actionArgs []interface{}) {
	isAlreadyMax = false
	optRange = optDomain
	actionArgs = nil

	if !i.TestDryRun() {
		return
	}

	// check if processing time is the main reason for latency
	processintTimeMetric := metrics.GetProcessingTimePerServiceMetricId(i.serviceId)
	value, ok := i.environment.GetMetric(processintTimeMetric)
	if !ok {
		log.Debugf("no value for metric %s", processintTimeMetric)
	} else {
		processingTime := value.(float64)

		clientLatencyMetric := metrics.GetClientLatencyPerServiceMetricId(i.serviceId)
		value, ok = i.environment.GetMetric(clientLatencyMetric)
		if !ok {
			log.Debugf("no value for metric %s", clientLatencyMetric)
			return
		}

		latency := value.(float64)

		processingTimePart := float32(processingTime) / float32(latency)
		if processingTimePart > processingThreshold {
			log.Debugf("most of the client latency is due to processing time (%f)", processingTimePart)
			isAlreadyMax = true
			return
		}
	}

	avgClientLocMetric := metrics.GetAverageClientLocationPerServiceMetricId(i.serviceId)
	value, ok = i.environment.GetMetric(avgClientLocMetric)
	if !ok {
		log.Debugf("no value for metric %s", avgClientLocMetric)
		return
	}

	// TODO change this for actual location
	avgClientLocation := value.(float64)

	candidateIds, sortingCriteria, ok := i.GenerateDomain(avgClientLocation)
	if !ok {
		return
	}

	log.Debugf("%s generated domain %+v", idealLatencyGoalId, candidateIds)
	filtered := i.Filter(candidateIds, optDomain)
	log.Debugf("%s filtered result %+v", idealLatencyGoalId, filtered)
	ordered := i.Order(filtered, sortingCriteria)
	log.Debugf("%s ordered result %+v", idealLatencyGoalId, ordered)

	optRange, isAlreadyMax = i.Cutoff(ordered, sortingCriteria)
	log.Debugf("%s cutoff result %+v", idealLatencyGoalId, optRange)

	furthestChild, _ := i.calcFurthestChildDistance(avgClientLocation)
	actionArgs = make([]interface{}, ilArgsNum, ilArgsNum)

	childMaxLoad := value.(float64)
	if childMaxLoad > loadToAddServiceThreshold {
		actionArgs[ilActionTypeArgIndex] = actions.ADD_SERVICE_ID
	} else {
		actionArgs[ilActionTypeArgIndex] = actions.MIGRATE_SERVICE_ID
		actionArgs[ilFromIndex] = furthestChild
	}

	return
}

func (i *idealLatency) GenerateDomain(arg interface{}) (domain goals.Domain, info map[string]interface{}, success bool) {
	value, ok := i.environment.GetMetric(metrics.MetricLocationInVicinity)
	if !ok {
		log.Debugf("no value for metric %s", metrics.MetricLocationInVicinity)
		return nil, nil, false
	}

	locationsInVicinity := value.(map[string]interface{})
	candidates := map[string]interface{}{}
	var candidateIds []string

	avgClientLocation := arg.(float64)
	_, furthestChildDistance := i.calcFurthestChildDistance(avgClientLocation)

	value, ok = i.environment.GetMetric(metrics.MetricNodeAddr)
	if !ok {
		log.Debugf("no value for metric %s", metrics.MetricNodeAddr)
		return nil, nil, false
	}

	myself := value.(string)

	log.Debugf("nodes in vicinity: %+v", locationsInVicinity)
	for nodeId, locationValue := range locationsInVicinity {
		_, okC := i.serviceChildren.Load(nodeId)
		_, okS := i.suspected.Load(nodeId)
		if okC || okS || nodeId == myself || nodeId == **i.parentId {
			log.Debugf("ignoring %s", nodeId)
			continue
		}
		location := locationValue.(float64)
		delta := math.Abs(location - avgClientLocation)
		candidates[nodeId] = &nodeWithDistance{
			NodeId:             nodeId,
			DistancePercentage: delta / furthestChildDistance,
		}
		candidateIds = append(candidateIds, nodeId)
	}

	return candidateIds, candidates, true
}

func (i *idealLatency) Order(candidates goals.Domain, sortingCriteria map[string]interface{}) (ordered goals.Range) {
	ordered = candidates
	sort.Slice(ordered, func(i, j int) bool {
		cI := sortingCriteria[ordered[i]].(*nodeWithDistance)
		cJ := sortingCriteria[ordered[j]].(*nodeWithDistance)

		return cI.DistancePercentage < cJ.DistancePercentage
	})

	return
}

func (i *idealLatency) Filter(candidates, domain goals.Domain) (filtered goals.Range) {
	return goals.DefaultFilter(candidates, domain)
}

func (i *idealLatency) Cutoff(candidates goals.Domain, candidatesCriteria map[string]interface{}) (cutoff goals.Range,
	maxed bool) {
	maxed = true

	for _, candidate := range candidates {
		percentage := candidatesCriteria[candidate].(*nodeWithDistance).DistancePercentage
		log.Debugf("candidate %s distance percentage from furthest child %f", candidate, percentage)
		if percentage < maximumDistancePercentage {
			cutoff = append(cutoff, candidate)
		}
		if percentage < equivalenDistancePercentage {
			maxed = false
		}
	}

	return
}

func (i *idealLatency) GenerateAction(target string, args ...interface{}) actions.Action {
	log.Debugf("generating action %s", (args[ilActionTypeArgIndex]).(string))

	switch args[ilActionTypeArgIndex].(string) {
	case actions.ADD_SERVICE_ID:
		return actions.NewAddServiceAction(i.serviceId, target)
	case actions.MIGRATE_SERVICE_ID:
		from := args[ilFromIndex].(string)
		return actions.NewMigrateAction(i.serviceId, from, target)
	}

	return nil
}

func (i *idealLatency) TestDryRun() (valid bool) {
	envCopy := i.environment.Copy()
	numInstancesMetric := metrics.GetNumInstancesMetricId(i.serviceId)
	value, ok := envCopy.GetMetric(numInstancesMetric)
	if !ok {
		log.Debugf("no value for metric %s", numInstancesMetric)
		return false
	}

	numInstances := value.(float64)
	envCopy.SetMetric(numInstancesMetric, numInstances+1)

	return envCopy.CheckConstraints() == nil
}

func (i *idealLatency) GetDependencies() (metrics []string) {
	return i.dependencies
}

func (i *idealLatency) calcFurthestChildDistance(avgLocation float64) (furthestChild string,
	furthestChildDistance float64) {
	furthestChildDistance = -1.0

	i.serviceChildren.Range(func(key, value interface{}) bool {
		childId := key.(serviceChildrenMapKey)
		child := value.(serviceChildrenMapValue)
		delta := child.Location - avgLocation

		if delta > furthestChildDistance {
			furthestChildDistance = delta
			furthestChild = childId
		}

		return true
	})

	if furthestChildDistance == -1.0 {
		log.Debugf("no furthest child")
		furthestChildDistance = math.MaxFloat64
	}

	return
}

func (i *idealLatency) GetId() string {
	return idealLatencyGoalId
}
