package service_goals

import (
	"sort"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/goals"
	log "github.com/sirupsen/logrus"
)

const (
	processingThreshold = 0.8

	maximumDistancePercentage   = 1.1
	equivalenDistancePercentage = 0.8
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

type NodeWithDistance struct {
	NodeId   string
	Distance float64
}

type IdealLatency struct {
	environment *autonomic.Environment
}

func NewIdealLatency(env *autonomic.Environment) *IdealLatency {
	return &IdealLatency{
		environment: env,
	}
}

func (i *IdealLatency) Optimize(optDomain goals.Domain) (isAlreadyMax bool, optRange goals.Range) {
	isAlreadyMax = false
	optRange = nil

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

	value, ok = i.environment.GetMetric(autonomic.METRIC_LOCATION)
	if !ok {
		log.Debugf("no value for metric %s", autonomic.METRIC_LOCATION)
		return
	}

	myLocation := value.(float64)

	value, ok = i.environment.GetMetric(autonomic.METRIC_AVERAGE_CLIENT_LOCATION)
	if !ok {
		log.Debugf("no value for metric %s", autonomic.METRIC_AVERAGE_CLIENT_LOCATION)
		return
	}

	// TODO change this for actual location
	avgClientLocation := value.(float64)

	candidateIds, candidates, ok := i.GenerateDomain(avgClientLocation)
	if !ok {
		return
	}

	filtered := i.Filter(candidateIds, optDomain)
	ordered := i.Order(filtered, candidates)

	myDelta := myLocation - avgClientLocation

	optRange, isAlreadyMax = i.Cutoff(ordered, myDelta, candidates)

	return
}

func (i *IdealLatency) Order(candidates goals.Domain, sortingCriteria map[string]interface{}) (ordered goals.Range) {
	ordered = candidates
	sort.Slice(ordered, func(i, j int) bool {
		cI := sortingCriteria[ordered[i]].(*NodeWithDistance)
		cJ := sortingCriteria[ordered[j]].(*NodeWithDistance)

		return cI.Distance < cJ.Distance
	})

	return
}

func (i *IdealLatency) Filter(candidates, domain goals.Domain) (filtered goals.Range) {
	return goals.DefaultFilter(candidates, domain)
}

func (i *IdealLatency) Cutoff(candidates goals.Domain, myCriteria interface{},
	candidatesCriteria map[string]interface{}) (cutoff goals.Range, maxed bool) {
	maxed = true

	for _, candidate := range candidates {
		percentage := candidatesCriteria[candidate].(*NodeWithDistance).Distance / myCriteria.(float64)
		if percentage < maximumDistancePercentage {
			cutoff = append(cutoff, candidate)
		}
		if percentage < equivalenDistancePercentage {
			maxed = false
		}
	}

	return
}

func (i *IdealLatency) GenerateDomain(arg interface{}) (domain goals.Domain, info map[string]interface{}, success bool) {
	value, ok := i.environment.GetMetric(autonomic.METRIC_LOCATION_IN_VICINITY)
	if !ok {
		log.Debugf("no value for metric %s", autonomic.METRIC_LOCATION_IN_VICINITY)
		return nil, nil, false
	}

	locationsInVicinity := value.(map[string]float64)
	candidates := map[string]interface{}{}
	var candidateIds []string

	for nodeId, location := range locationsInVicinity {
		delta := location - arg.(float64)
		candidates[nodeId] = &NodeWithDistance{
			NodeId:   nodeId,
			Distance: delta,
		}
		candidateIds = append(candidateIds, nodeId)
	}

	return candidateIds, candidates, true
}

func (i *IdealLatency) GenerateAction(target string) actions.Action {
	return actions.NewMigrateAction(target)
}

func (i *IdealLatency) TestDryRun() (valid bool) {
	envCopy := i.environment.Copy()
	value, ok := envCopy.GetMetric(autonomic.METRIC_NUMBER_OF_INSTANCES_ID)
	if !ok {
		return false
	}

	numInstances := value.(int)
	envCopy.SetMetric(autonomic.METRIC_NUMBER_OF_INSTANCES_ID, numInstances+1)

	return envCopy.CheckConstraints() == nil
}

func (i *IdealLatency) GetDependencies() (metrics []string) {
	return idealLatencyDependencies
}
