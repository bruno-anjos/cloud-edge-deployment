package service

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/metrics"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	publicUtils "github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
)

const (
	processingThreshold = 0.8

	maximumDistancePercentage = 1.2
	satisfiedDistance         = 200.
	maxDistance               = 5000.
	maxChildren               = 3.
	branchingCutoff           = 1

	idealLatencyGoalId = "GOAL_IDEAL_LATENCY"

	hiddenParentId = "_parent"
)

const (
	ilActionTypeArgIndex = iota
	ilFromIndex
	ilExploreIndex
	ilArgsNum
)

type (
	serviceChildrenMapKey   = string
	serviceChildrenMapValue = *nodeWithLocation
)

type nodeWithDistance struct {
	NodeId             string
	DistancePercentage float64
}

type idealLatency struct {
	service      *Service
	dependencies []string
}

func newIdealLatencyGoal(service *Service) *idealLatency {

	dependencies := []string{
		metrics.GetProcessingTimePerServiceMetricId(service.ServiceId),
		metrics.GetClientLatencyPerServiceMetricId(service.ServiceId),
		metrics.MetricLocation,
		metrics.GetAverageClientLocationPerServiceMetricId(service.ServiceId),
		metrics.MetricLocationInVicinity,
		metrics.GetNumInstancesMetricId(service.ServiceId),
	}

	goal := &idealLatency{
		service:      service,
		dependencies: dependencies,
	}

	return goal
}

func (i *idealLatency) Optimize(optDomain Domain) (isAlreadyMax bool, optRange Range,
	actionArgs []interface{}) {
	isAlreadyMax = true
	optRange = optDomain
	actionArgs = nil

	if !i.TestDryRun() {
		return
	}

	// check if processing time is the main reason for latency
	processingTimeTooHigh := i.checkProcessingTime()
	if processingTimeTooHigh {
		return
	}

	archClient := archimedes.NewArchimedesClient("localhost:" + strconv.Itoa(archimedes.Port))
	avgClientLocation, status := archClient.GetAvgClientLocation(i.service.ServiceId)
	if status == http.StatusNoContent {
		return
	} else if status != http.StatusOK {
		log.Errorf("got status code %d while attempting to get avg client location for service %s", status,
			i.service.ServiceId)
		return
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

	var (
		numChildren  int
		shouldBranch bool
	)

	shouldBranch, numChildren = i.checkShouldBranch(avgClientLocation)
	isAlreadyMax = !shouldBranch

	if !isAlreadyMax {
		optRange, isAlreadyMax = i.filterBlacklisted(optRange)
		if !isAlreadyMax {
			actionArgs = make([]interface{}, ilArgsNum, ilArgsNum)
			actionArgs[ilActionTypeArgIndex] = actions.ExtendServiceId
			actionArgs[ilExploreIndex] = numChildren > 0
		}
	}

	return
}

func (i *idealLatency) GenerateDomain(arg interface{}) (domain Domain, info map[string]interface{}, success bool) {
	value, ok := i.service.Environment.GetMetric(metrics.MetricLocationInVicinity)
	if !ok {
		log.Debugf("no value for metric %s", metrics.MetricLocationInVicinity)
		return nil, nil, false
	}

	locationsInVicinity := value.(map[string]interface{})
	candidates := map[string]interface{}{}
	var candidateIds []string

	var avgClientLocation publicUtils.Location
	err := mapstructure.Decode(arg, &avgClientLocation)
	if err != nil {
		panic(err)
	}

	value, ok = i.service.Environment.GetMetric(metrics.MetricLocation)
	if !ok {
		log.Fatalf("no value for metric %s", metrics.MetricNodeAddr)
	}

	var location publicUtils.Location
	err = mapstructure.Decode(value, &location)
	if err != nil {
		panic(err)
	}

	myDist := avgClientLocation.CalcDist(&location)

	value, ok = i.service.Environment.GetMetric(metrics.MetricNodeAddr)
	if !ok {
		log.Debugf("no value for metric %s", metrics.MetricNodeAddr)
		return nil, nil, false
	}

	log.Debugf("nodes in vicinity: %+v", locationsInVicinity)
	for nodeId, locationValue := range locationsInVicinity {
		_, okC := i.service.Children.Load(nodeId)
		_, okS := i.service.Suspected.Load(nodeId)
		if okC || okS || nodeId == myself.Id {
			log.Debugf("ignoring %s", nodeId)
			continue
		}

		err = mapstructure.Decode(locationValue, &location)
		if err != nil {
			panic(err)
		}

		delta := location.CalcDist(&avgClientLocation)

		if nodeId == i.service.ParentId {
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

func (i *idealLatency) Order(candidates Domain, sortingCriteria map[string]interface{}) (ordered Range) {
	ordered = candidates
	sort.Slice(ordered, func(i, j int) bool {
		cI := sortingCriteria[ordered[i]].(*nodeWithDistance)
		cJ := sortingCriteria[ordered[j]].(*nodeWithDistance)
		return cI.DistancePercentage < cJ.DistancePercentage
	})

	return
}

func (i *idealLatency) Filter(candidates, domain Domain) (filtered Range) {
	return DefaultFilter(candidates, domain)
}

func (i *idealLatency) Cutoff(candidates Domain, candidatesCriteria map[string]interface{}) (cutoff Range,
	maxed bool) {
	maxed = true

	avgClientLocMetric := metrics.GetAverageClientLocationPerServiceMetricId(i.service.ServiceId)
	value, ok := i.service.Environment.GetMetric(avgClientLocMetric)
	if !ok {
		log.Debugf("no value for metric %s", avgClientLocMetric)
		return
	}

	// TODO change this for actual location
	var avgClientLocation publicUtils.Location
	err := mapstructure.Decode(value, &avgClientLocation)
	if err != nil {
		panic(err)
	}

	candidateClient := deployer.NewDeployerClient("")
	for _, candidate := range candidates {
		percentage := candidatesCriteria[candidate].(*nodeWithDistance).DistancePercentage
		log.Debugf("candidate %s distance percentage (me) %f", candidate, percentage)
		if percentage < maximumDistancePercentage {
			candidateClient.SetHostPort(candidate + ":" + strconv.Itoa(deployer.Port))
			has, _ := candidateClient.HasService(i.service.ServiceId)
			if has {
				log.Debugf("candidate %s already has service %s", candidate, i.service.ServiceId)
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
	case actions.ExtendServiceId:
		return actions.NewExtendServiceAction(i.service.ServiceId, target, args[ilExploreIndex].(bool),
			myself, nil)
	case actions.MigrateServiceId:
		from := args[ilFromIndex].(string)
		return actions.NewMigrateAction(i.service.ServiceId, from, target)
	}

	return nil
}

func (i *idealLatency) TestDryRun() (valid bool) {
	envCopy := i.service.Environment.Copy()
	numInstancesMetric := metrics.GetNumInstancesMetricId(i.service.ServiceId)
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

func (i *idealLatency) calcFurthestChildDistance(avgLocation *publicUtils.Location) (furthestChild string,
	furthestChildDistance float64) {
	furthestChildDistance = -1.0

	i.service.Children.Range(func(key, value interface{}) bool {
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
		value, ok := i.service.Environment.GetMetric(metrics.MetricNodeAddr)
		if !ok {
			log.Fatalf("no value for metric %s", metrics.MetricNodeAddr)
		}

		value, ok = i.service.Environment.GetMetric(metrics.MetricLocation)
		if !ok {
			log.Fatalf("no value for metric %s", metrics.MetricNodeAddr)
		}

		var location publicUtils.Location
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
	processintTimeMetric := metrics.GetProcessingTimePerServiceMetricId(i.service.ServiceId)
	value, ok := i.service.Environment.GetMetric(processintTimeMetric)
	if !ok {
		log.Debugf("no value for metric %s", processintTimeMetric)
	} else {
		processingTime := value.(float64)

		clientLatencyMetric := metrics.GetClientLatencyPerServiceMetricId(i.service.ServiceId)
		value, ok = i.service.Environment.GetMetric(clientLatencyMetric)
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

func (i *idealLatency) checkShouldBranch(avgClientLocation *publicUtils.Location) (bool, int) {
	numChildren := 0
	i.service.Children.Range(func(key, value interface{}) bool {
		numChildren++
		return true
	})

	value, ok := i.service.Environment.GetMetric(metrics.MetricLocation)
	if !ok {
		log.Fatalf("no value for metric %s", metrics.MetricNodeAddr)
	}

	var location publicUtils.Location
	err := mapstructure.Decode(value, &location)
	if err != nil {
		panic(err)
	}

	currDistance := location.CalcDist(avgClientLocation)
	if currDistance <= satisfiedDistance {
		return false, numChildren
	}

	// TODO this has to be tuned for real distances
	distanceFactor := maxDistance / (maxDistance - (currDistance - satisfiedDistance))
	childrenFactor := (((maxChildren + 1.) / (float64(numChildren) + 1.)) - 1.) / maxChildren
	branchingFactor := childrenFactor * distanceFactor
	log.Debugf("branching factor %f (%d)", branchingFactor, numChildren)

	validBranch := branchingFactor > branchingCutoff
	log.Debugf("should branch: %t", validBranch)

	return validBranch, numChildren
}

func (i *idealLatency) filterBlacklisted(o Range) (Range, bool) {
	var newRange []string
	for _, node := range o {
		if _, ok := i.service.Blacklist.Load(node); !ok {
			newRange = append(newRange, node)
		}
	}

	log.Debugf("after filtering blacklisted: %+v", newRange)

	return newRange, len(newRange) == 0
}
