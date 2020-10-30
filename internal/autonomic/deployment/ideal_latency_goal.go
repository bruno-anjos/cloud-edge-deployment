package deployment

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/metrics"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/golang/geo/s1"
	"github.com/golang/geo/s2"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
)

const (
	processingThreshold = 0.8

	maximumDistancePercentage = 1.2
	satisfiedDistance         = 20.
	maxDistance               = 5000.
	maxChildren               = 4.
	branchingCutoff           = 1

	idealLatencyGoalId = "GOAL_IDEAL_LATENCY"

	hiddenParentId = "_parent"
)

const (
	ilActionTypeArgIndex = iota
	ilFromIndex
	ilExploreNodesIndex
	ilExploreLocationIndex
	ilArgsNum
)

type (
	deploymentChildrenMapKey   = string
	deploymentChildrenMapValue = *nodeWithLocation

	nodeWithDistance struct {
		NodeId             string
		DistancePercentage float64
	}

	sortingCriteriaType = map[s2.CellID]*nodeWithDistance

	generateDomainArg struct {
		CentroidCells map[s2.CellID]s2.Cell
		MyDists       map[s2.CellID]s1.ChordAngle
	}
)

type idealLatency struct {
	deployment   *Deployment
	myLocation   s2.CellID
	dependencies []string
}

func newIdealLatencyGoal(deployment *Deployment) *idealLatency {

	dependencies := []string{
		metrics.GetProcessingTimePerDeploymentMetricId(deployment.DeploymentId),
		metrics.GetClientLatencyPerDeploymentMetricId(deployment.DeploymentId),
		metrics.MetricLocation,
		metrics.GetAverageClientLocationPerDeploymentMetricId(deployment.DeploymentId),
		metrics.MetricLocationInVicinity,
		metrics.GetNumInstancesMetricId(deployment.DeploymentId),
	}

	value, ok := deployment.Environment.GetMetric(metrics.MetricLocation)
	if !ok {
		panic("could not get location from environment")
	}

	goal := &idealLatency{
		deployment:   deployment,
		dependencies: dependencies,
		myLocation:   value.(s2.CellID),
	}

	return goal
}

func (i *idealLatency) Optimize(optDomain Domain) (isAlreadyMax bool, optRange Range,
	actionArgs []interface{}) {
	isAlreadyMax = true
	optRange = optDomain
	actionArgs = nil

	// check if processing time is the main reason for latency
	processingTimeTooHigh := i.checkProcessingTime()
	if processingTimeTooHigh {
		return
	}

	archClient := archimedes.NewArchimedesClient("localhost:" + strconv.Itoa(archimedes.Port))
	centroids, status := archClient.GetClientCentroids(i.deployment.DeploymentId)
	if status == http.StatusNoContent {
		return
	} else if status != http.StatusOK {
		log.Errorf("got status code %d while attempting to get client centroids for deployment %s", status,
			i.deployment.DeploymentId)
		return
	}

	candidateIds, sortingCriteria, ok := i.GenerateDomain(centroids)
	if !ok {
		return
	}

	log.Debugf("%s generated domain %+v", idealLatencyGoalId, candidateIds)
	filtered := i.Filter(candidateIds, optDomain)

	nodeMinDistances := map[string]interface{}{}
	for _, node := range filtered {
		var (
			minPercentage = -1.
			minCellId     s2.CellID
		)
		for cellId, criteria := range sortingCriteria[node].(sortingCriteriaType) {
			if criteria.DistancePercentage < minPercentage || minPercentage == -1 {
				minPercentage = criteria.DistancePercentage
				minCellId = cellId
			}
		}

		nodeMinDistances[node] = minPercentage
	}

	log.Debugf("%s filtered result %+v", idealLatencyGoalId, filtered)
	ordered := i.Order(filtered, nodeMinDistances)
	log.Debugf("%s ordered result %+v", idealLatencyGoalId, ordered)

	optRange, isAlreadyMax = i.Cutoff(ordered, nodeMinDistances)
	log.Debugf("%s cutoff result (%t) %+v", idealLatencyGoalId, isAlreadyMax, optRange)

	if len(optRange) == 0 {
		return
	}

	// var (
	// 	shouldBranch bool
	// )
	//
	// shouldBranch = i.checkShouldBranch(avgClientLocation)
	// isAlreadyMax = !shouldBranch

	if !isAlreadyMax {
		optRange, isAlreadyMax = i.filterBlacklisted(optRange)
		if !isAlreadyMax {
			actionArgs = make([]interface{}, ilArgsNum, ilArgsNum)
			actionArgs[ilActionTypeArgIndex] = actions.ExtendDeploymentId
			exploreNodes := map[string]interface{}{}
			for _, nodeId := range optRange {
				if sortingCriteria[nodeId].(sortingCriteriaType).DistancePercentage > 1. {
					exploreNodes[nodeId] = nil
				}
			}
			actionArgs[ilExploreNodesIndex] = exploreNodes
			actionArgs[ilExploreLocationIndex] = avgClientLocation
		}
	}

	return
}

func (i *idealLatency) GenerateDomain(arg interface{}) (domain Domain, info map[string]interface{}, success bool) {
	value, ok := i.deployment.Environment.GetMetric(metrics.MetricLocationInVicinity)
	if !ok {
		log.Debugf("no value for metric %s", metrics.MetricLocationInVicinity)
		return nil, nil, false
	}

	locationsInVicinity := value.(map[string]interface{})

	centroids := arg.([]s2.CellID)

	var (
		myDists       = map[s2.CellID]s1.ChordAngle{}
		centroidCells = map[s2.CellID]s2.Cell{}
	)
	for _, centroid := range centroids {
		centroidCell := s2.CellFromCellID(centroid)
		myDists[centroid] = centroidCell.DistanceToCell(s2.CellFromCellID(i.myLocation))
		centroidCells[centroid] = centroidCell
	}

	log.Debugf("nodes in vicinity: %+v", locationsInVicinity)
	info = map[string]interface{}{}
	for nodeId, cellValue := range locationsInVicinity {
		_, okC := i.deployment.Children.Load(nodeId)
		_, okS := i.deployment.Suspected.Load(nodeId)
		if okC || okS || nodeId == myself.Id {
			log.Debugf("ignoring %s", nodeId)
			continue
		}

		location := s2.CellIDFromToken(cellValue.(string))

		// create node map for centroids and respective distances
		if nodeId == i.deployment.ParentId {
			info[hiddenParentId] = sortingCriteriaType{}
		} else {
			info[nodeId] = sortingCriteriaType{}
			domain = append(domain, nodeId)
		}

		var (
			nodeCentroidsMap sortingCriteriaType
		)
		for _, centroidId := range centroids {
			delta := s2.CellFromCellID(location).DistanceToCell(centroidCells[centroidId])

			if nodeId == i.deployment.ParentId {
				nodeCentroidsMap = info[hiddenParentId].(sortingCriteriaType)
			} else {
				nodeCentroidsMap = info[nodeId].(sortingCriteriaType)
			}
			nodeCentroidsMap[centroidId] = &nodeWithDistance{
				NodeId:             nodeId,
				DistancePercentage: float64(delta) / float64(myDists[centroidId]),
			}
		}
	}

	success = true
	return
}

func (i *idealLatency) Order(candidates Domain, sortingCriteria map[string]interface{}) (ordered Range) {
	ordered = candidates
	sort.Slice(ordered, func(i, j int) bool {
		return sortingCriteria[ordered[i]].(float64) < sortingCriteria[ordered[j]].(float64)
	})

	return
}

func (i *idealLatency) Filter(candidates, domain Domain) (filtered Range) {
	return DefaultFilter(candidates, domain)
}

func (i *idealLatency) Cutoff(candidates Domain, candidatesCriteria map[string]interface{}) (cutoff Range,
	maxed bool) {
	maxed = true

	candidateClient := deployer.NewDeployerClient("")
	for _, candidate := range candidates {
		percentage := candidatesCriteria[candidate].(float64)
		log.Debugf("candidate %s distance percentage (me) %f", candidate, percentage)
		if percentage < maximumDistancePercentage {
			candidateClient.SetHostPort(candidate + ":" + strconv.Itoa(deployer.Port))
			has, _ := candidateClient.HasDeployment(i.deployment.DeploymentId)
			if has {
				log.Debugf("candidate %s already has deployment %s", candidate, i.deployment.DeploymentId)
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
	case actions.ExtendDeploymentId:
		_, exploring := args[ilExploreNodesIndex].(map[string]interface{})[target]
		return actions.NewExtendDeploymentAction(i.deployment.DeploymentId, target, exploring, myself, nil,
			args[ilExploreLocationIndex].(s2.CellID))
	case actions.MigrateDeploymentId:
		from := args[ilFromIndex].(string)
		return actions.NewMigrateAction(i.deployment.DeploymentId, from, target)
	}

	return nil
}

func (i *idealLatency) GetDependencies() (metrics []string) {
	return i.dependencies
}

func (i *idealLatency) calcFurthestChildDistance(avgLocation s2.CellID) (furthestChild string,
	furthestChildDistance s1.ChordAngle) {
	furthestChildDistance = -1.0

	i.deployment.Children.Range(func(key, value interface{}) bool {
		childId := key.(deploymentChildrenMapKey)
		child := value.(deploymentChildrenMapValue)
		delta := s2.CellFromCellID(child.Location).DistanceToCell(s2.CellFromCellID(avgLocation))

		if delta > furthestChildDistance {
			furthestChildDistance = delta
			furthestChild = childId
		}

		log.Debugf("child %s", childId)

		return true
	})

	if furthestChildDistance == -1.0 {
		value, ok := i.deployment.Environment.GetMetric(metrics.MetricNodeAddr)
		if !ok {
			log.Fatalf("no value for metric %s", metrics.MetricNodeAddr)
		}

		value, ok = i.deployment.Environment.GetMetric(metrics.MetricLocation)
		if !ok {
			log.Fatalf("no value for metric %s", metrics.MetricNodeAddr)
		}

		var location s2.CellID
		err := mapstructure.Decode(value, &location)
		if err != nil {
			panic(err)
		}

		furthestChildDistance = s2.CellFromCellID(location).DistanceToCell(s2.CellFromCellID(avgLocation))
	}

	return
}

func (i *idealLatency) GetId() string {
	return idealLatencyGoalId
}

func (i *idealLatency) checkProcessingTime() bool {
	processintTimeMetric := metrics.GetProcessingTimePerDeploymentMetricId(i.deployment.DeploymentId)
	value, ok := i.deployment.Environment.GetMetric(processintTimeMetric)
	if !ok {
		log.Debugf("no value for metric %s", processintTimeMetric)
	} else {
		processingTime := value.(float64)

		clientLatencyMetric := metrics.GetClientLatencyPerDeploymentMetricId(i.deployment.DeploymentId)
		value, ok = i.deployment.Environment.GetMetric(clientLatencyMetric)
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

func (i *idealLatency) checkShouldBranch(centroids []) bool {
	numChildren := 0
	i.deployment.Children.Range(func(key, value interface{}) bool {
		numChildren++
		return true
	})

	value, ok := i.deployment.Environment.GetMetric(metrics.MetricLocation)
	if !ok {
		log.Fatalf("no value for metric %s", metrics.MetricNodeAddr)
	}

	var location s2.CellID
	err := mapstructure.Decode(value, &location)
	if err != nil {
		panic(err)
	}

	currDistance := utils.ChordAngleToKM(s2.CellFromCellID(location).
		DistanceToCell(s2.CellFromCellID(avgClientLocation)))
	if currDistance <= satisfiedDistance {
		return false
	}

	distanceFactor := maxDistance / (maxDistance - (currDistance - satisfiedDistance))
	childrenFactor := (((maxChildren + 1.) / (float64(numChildren) + 1.)) - 1.) / maxChildren
	branchingFactor := childrenFactor * distanceFactor
	log.Debugf("branching factor %f (%d)", branchingFactor, numChildren)

	validBranch := branchingFactor > branchingCutoff
	log.Debugf("should branch: %t", validBranch)

	return validBranch
}

func (i *idealLatency) filterBlacklisted(o Range) (Range, bool) {
	var newRange []string
	for _, node := range o {
		if _, ok := i.deployment.Blacklist.Load(node); !ok {
			newRange = append(newRange, node)
		}
	}

	log.Debugf("after filtering blacklisted: %+v", newRange)

	return newRange, len(newRange) == 0
}
