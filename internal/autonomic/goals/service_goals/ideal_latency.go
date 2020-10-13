package service_goals

import (
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/environment"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/goals"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/metrics"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
)

const (
	processingThreshold = 0.8

	maximumDistancePercentage   = 1.4
	equivalenDistancePercentage = 0.8
	loadToAddServiceThreshold   = 0.7

	ilArgsNum = 3

	idealLatencyGoalId = "GOAL_IDEAL_LATENCY"

	hiddenParentId = "_parent"

	blacklistDuration = 5
)

const (
	ilActionTypeArgIndex = iota
	ilFromIndex
	ilExploreIndex
)

type NodeWithLocation struct {
	NodeId   string
	Location *utils.Location
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
	parentId        *string
	exploring       sync.Map
	blacklist       sync.Map
}

func NewIdealLatency(serviceId string, serviceChildren, suspected *sync.Map,
	parentId *string, env *environment.Environment) *idealLatency {
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
	processingTimeTooHigh := i.checkProcessingTime()
	if processingTimeTooHigh {
		isAlreadyMax = true
		return
	}

	avgClientLocMetric := metrics.GetAverageClientLocationPerServiceMetricId(i.serviceId)
	value, ok := i.environment.GetMetric(avgClientLocMetric)
	if !ok {
		log.Debugf("no value for metric %s", avgClientLocMetric)
		return
	}

	var avgClientLocation utils.Location
	err := mapstructure.Decode(value, &avgClientLocation)
	if err != nil {
		panic(err)
	}

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
	log.Debugf("%s cutoff result (%t) %+v", idealLatencyGoalId, isAlreadyMax, optRange)

	if len(optRange) == 0 {
		return
	}

	isAlreadyMax = !i.checkShouldBranch(&avgClientLocation)

	if !isAlreadyMax {
		actionArgs = make([]interface{}, ilArgsNum, ilArgsNum)

		exploring := sortingCriteria[optRange[0]].(*nodeWithDistance).DistancePercentage > 1.
		if exploring {
			exploreChan := make(chan struct{})
			go i.waitToBlacklist(optRange[0], exploreChan)
			actionArgs[ilExploreIndex] = exploreChan
		}

		actionArgs[ilActionTypeArgIndex] = actions.AddServiceId
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

	var avgClientLocation utils.Location
	err := mapstructure.Decode(arg, &avgClientLocation)
	if err != nil {
		panic(err)
	}

	value, ok = i.environment.GetMetric(metrics.MetricLocation)
	if !ok {
		log.Fatalf("no value for metric %s", metrics.MetricNodeAddr)
	}

	var location utils.Location
	err = mapstructure.Decode(value, &location)
	if err != nil {
		panic(err)
	}

	myDist := avgClientLocation.CalcDist(&location)

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
		_, okB := i.blacklist.Load(nodeId)
		if okC || okS || okB || nodeId == myself {
			log.Debugf("ignoring %s", nodeId)
			continue
		}

		err = mapstructure.Decode(locationValue, &location)
		if err != nil {
			panic(err)
		}

		delta := location.CalcDist(&avgClientLocation)

		if nodeId == *i.parentId {
			candidates[hiddenParentId] = &nodeWithDistance{
				NodeId:             nodeId,
				DistancePercentage: delta / myDist,
			}
		} else {
			candidates[nodeId] = &nodeWithDistance{
				NodeId:             nodeId,
				DistancePercentage: delta / myDist,
			}
			candidateIds = append(candidateIds, nodeId)
		}
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

	avgClientLocMetric := metrics.GetAverageClientLocationPerServiceMetricId(i.serviceId)
	value, ok := i.environment.GetMetric(avgClientLocMetric)
	if !ok {
		log.Debugf("no value for metric %s", avgClientLocMetric)
		return
	}

	// TODO change this for actual location
	var avgClientLocation utils.Location
	err := mapstructure.Decode(value, &avgClientLocation)
	if err != nil {
		panic(err)
	}

	for _, candidate := range candidates {
		percentage := candidatesCriteria[candidate].(*nodeWithDistance).DistancePercentage
		log.Debugf("candidate %s distance percentage from furthest child %f", candidate, percentage)
		if percentage < maximumDistancePercentage {
			candidateClient.SetHostPort(candidate + ":" + strconv.Itoa(deployer.Port))
			has, _ := candidateClient.HasService(i.serviceId)
			if has {
				log.Debugf("candidate %s already has service %s", candidate, i.serviceId)
				continue
			}
			cutoff = append(cutoff, candidate)
		}
		if percentage < 1. {
			maxed = false
		}
	}

	return
}

func (i *idealLatency) GenerateAction(target string, args ...interface{}) actions.Action {
	log.Debugf("generating action %s", (args[ilActionTypeArgIndex]).(string))

	switch args[ilActionTypeArgIndex].(string) {
	case actions.AddServiceId:
		var exploreChan chan struct{}
		if args[ilExploreIndex] != nil {
			exploreChan = args[ilExploreIndex].(chan struct{})
		}
		return actions.NewAddServiceAction(i.serviceId, target, exploreChan)
	case actions.MigrateServiceId:
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

func (i *idealLatency) calcFurthestChildDistance(avgLocation *utils.Location) (furthestChild string,
	furthestChildDistance float64) {
	furthestChildDistance = -1.0

	i.serviceChildren.Range(func(key, value interface{}) bool {
		childId := key.(serviceChildrenMapKey)
		child := value.(serviceChildrenMapValue)
		delta := child.Location.CalcDist(avgLocation)

		if delta > furthestChildDistance {
			furthestChildDistance = delta
			furthestChild = childId
		}

		log.Debugf("child %s", childId)

		return true
	})

	if furthestChildDistance == -1.0 {
		value, ok := i.environment.GetMetric(metrics.MetricNodeAddr)
		if !ok {
			log.Fatalf("no value for metric %s", metrics.MetricNodeAddr)
		}

		value, ok = i.environment.GetMetric(metrics.MetricLocation)
		if !ok {
			log.Fatalf("no value for metric %s", metrics.MetricNodeAddr)
		}

		var location utils.Location
		err := mapstructure.Decode(value, &location)
		if err != nil {
			panic(err)
		}

		furthestChildDistance = location.CalcDist(avgLocation)
	}

	return
}

func (i *idealLatency) GetId() string {
	return idealLatencyGoalId
}

func (i *idealLatency) checkProcessingTime() bool {
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
			return true
		}

		latency := value.(float64)

		processingTimePart := float32(processingTime) / float32(latency)
		if processingTimePart > processingThreshold {
			log.Debugf("most of the client latency is due to processing time (%f)", processingTimePart)
			return true
		}
	}

	return false
}

func (i *idealLatency) checkShouldBranch(avgClientLocation *utils.Location) bool {
	numChildren := 0
	i.serviceChildren.Range(func(key, value interface{}) bool {
		numChildren++
		return true
	})

	value, ok := i.environment.GetMetric(metrics.MetricLocation)
	if !ok {
		log.Fatalf("no value for metric %s", metrics.MetricNodeAddr)
	}

	var location utils.Location
	err := mapstructure.Decode(value, &location)
	if err != nil {
		panic(err)
	}

	currDistance := location.CalcDist(avgClientLocation)

	// TODO this has to be tuned for real distances
	branchingFactor := ((1 / (currDistance / 500)) * 100) + (20 / float64(numChildren))

	// TODO these values seem to make sense for now
	branch := (branchingFactor / float64(numChildren+1)) > 20

	log.Debugf("branching factor %f (%d)", branchingFactor, numChildren)
	log.Debugf("should i branch? %t", branch)

	return branch
}

func (i *idealLatency) waitToBlacklist(childId string, exploredChan <-chan struct{}) {
	interval := (4 * 30) * time.Second

	timer := time.NewTimer(interval)
	select {
	case <-exploredChan:
		log.Debugf("exploring %s through %s was a success", i.serviceId, childId)
		return
	case <-timer.C:
		log.Debugf("blacklisting %s", childId)
		i.blacklist.Store(childId, nil)
		blacklistTimer := time.NewTimer(blacklistDuration * time.Minute)
		<-blacklistTimer.C
		log.Debugf("removing %s from blacklist", childId)
		i.blacklist.Delete(childId)
	}
}
