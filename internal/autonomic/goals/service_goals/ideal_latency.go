package service_goals

import (
	"math"
	"sort"
	"sync"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/goals"
	log "github.com/sirupsen/logrus"
)

const (
	processingThreshold = 0.8

	maximumDistancePercentage   = 1.1
	equivalenDistancePercentage = 0.8
	loadToAddServiceThreshold   = 0.7

	ilArgsNum = 2

	ilActionTypeArgIndex = iota
	ilFromIndex
)

var (
	idealLatencyDependencies = []string{
		autonomic.METRIC_PROCESSING_TIME_PER_SERVICE,
		autonomic.METRIC_CLIENT_LATENCY_PER_SERVICE,
		autonomic.METRIC_LOCATION,
		autonomic.METRIC_AVERAGE_CLIENT_LOCATION,
		autonomic.METRIC_LOCATION_IN_VICINITY,
		autonomic.METRIC_NUMBER_OF_INSTANCES_ID,
	}
)

type (
	serviceChildrenMapKey   = string
	serviceChildrenMapValue = *nodeWithLocation
)

type nodeWithLocation struct {
	NodeId   string
	Location float64
}

type nodeWithDistance struct {
	NodeId             string
	DistancePercentage float64
}

type idealLatency struct {
	serviceId       string
	serviceChildren *sync.Map
	environment     *autonomic.Environment
}

func NewIdealLatency(serviceId string, serviceChildren *sync.Map, env *autonomic.Environment) *idealLatency {
	goal := &idealLatency{
		serviceId:       serviceId,
		serviceChildren: serviceChildren,
		environment:     env,
	}

	return goal
}

func (i *idealLatency) Optimize(optDomain goals.Domain) (isAlreadyMax bool, optRange goals.Range,
	actionArgs []interface{}) {
	isAlreadyMax = false
	optRange = nil
	actionArgs = nil

	if !i.TestDryRun() {
		return
	}

	// check if processing time is the main reason for latency
	value, ok := i.environment.GetMetric(autonomic.METRIC_PROCESSING_TIME_PER_SERVICE)
	if !ok {
		log.Debugf("no value for metric %s", autonomic.METRIC_PROCESSING_TIME_PER_SERVICE)
	} else {
		processingTime := value.(int)

		value, ok = i.environment.GetMetric(autonomic.METRIC_CLIENT_LATENCY_PER_SERVICE)
		if !ok {
			log.Debugf("no value for metric %s", autonomic.METRIC_CLIENT_LATENCY_PER_SERVICE)
			return
		}

		latency := value.(int)

		processingTimePart := float32(processingTime) / float32(latency)
		if processingTimePart > processingThreshold {
			log.Debugf("most of the client latency is due to processing time (%f)", processingTimePart)
			isAlreadyMax = true
			return
		}
	}

	value, ok = i.environment.GetMetric(autonomic.METRIC_AVERAGE_CLIENT_LOCATION)
	if !ok {
		log.Debugf("no value for metric %s", autonomic.METRIC_AVERAGE_CLIENT_LOCATION)
		return
	}

	// TODO change this for actual location
	avgClientLocation := value.(float64)

	candidateIds, sortingCriteria, ok := i.GenerateDomain(avgClientLocation)
	if !ok {
		return
	}

	filtered := i.Filter(candidateIds, optDomain)
	ordered := i.Order(filtered, sortingCriteria)

	optRange, isAlreadyMax = i.Cutoff(ordered, sortingCriteria)

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
		if percentage < maximumDistancePercentage {
			cutoff = append(cutoff, candidate)
		}
		if percentage < equivalenDistancePercentage {
			maxed = false
		}
	}

	return
}

func (i *idealLatency) GenerateDomain(arg interface{}) (domain goals.Domain, info map[string]interface{}, success bool) {
	value, ok := i.environment.GetMetric(autonomic.METRIC_LOCATION_IN_VICINITY)
	if !ok {
		log.Debugf("no value for metric %s", autonomic.METRIC_LOCATION_IN_VICINITY)
		return nil, nil, false
	}

	locationsInVicinity := value.(map[string]float64)
	candidates := map[string]interface{}{}
	var candidateIds []string

	avgClientLocation := arg.(float64)
	_, furthestChildDistance := i.calcFurthestChildDistance(avgClientLocation)

	for nodeId, location := range locationsInVicinity {
		_, ok = i.serviceChildren.Load(nodeId)
		if ok {
			continue
		}
		delta := location - avgClientLocation
		candidates[nodeId] = &nodeWithDistance{
			NodeId:             nodeId,
			DistancePercentage: delta / furthestChildDistance,
		}
		candidateIds = append(candidateIds, nodeId)
	}

	return candidateIds, candidates, true
}

func (i *idealLatency) GenerateAction(target string, args ...interface{}) actions.Action {
	switch args[ilActionTypeArgIndex] {
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
	value, ok := envCopy.GetMetric(autonomic.METRIC_NUMBER_OF_INSTANCES_ID)
	if !ok {
		return false
	}

	numInstances := value.(int)
	envCopy.SetMetric(autonomic.METRIC_NUMBER_OF_INSTANCES_ID, numInstances+1)

	return envCopy.CheckConstraints() == nil
}

func (i *idealLatency) GetDependencies() (metrics []string) {
	return idealLatencyDependencies
}

func (i *idealLatency) calcFurthestChildDistance(avgLocation float64) (
	furthestChild string, furthestChildDistance float64) {
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
		furthestChildDistance = math.MaxFloat64
	}

	return
}
