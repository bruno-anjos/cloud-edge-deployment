package deployment

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"

	deployerAPI "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/metrics"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/golang/geo/s2"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
)

const (
	processingThreshold = 0.8

	maximumDistancePercentage = 1.2
	satisfiedDistance         = 20.
	maxDistance               = utils.EarthRadius * math.Pi
	maxChildren               = 4.
	branchingCutoff           = 1

	maxExploringTTL = 3

	idealLatencyGoalId = "GOAL_IDEAL_LATENCY"

	hiddenParentId = "_parent"
)

const (
	ilActionTypeArgIndex = iota
	ilCentroidDistancesToNodesIndex
	ilExploringCentroidsIndex
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
)

type idealLatency struct {
	deployment        *Deployment
	myLocation        s2.CellID
	dependencies      []string
	centroidsExtended map[s2.CellID]interface{}
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
		deployment:        deployment,
		dependencies:      dependencies,
		myLocation:        s2.CellIDFromToken(value.(string)),
		centroidsExtended: map[s2.CellID]interface{}{},
	}

	return goal
}

func (i *idealLatency) Optimize(optDomain Domain) (isAlreadyMax bool, optRange Range, actionArgs []interface{}) {
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
		)
		for _, criteria := range sortingCriteria[node].(sortingCriteriaType) {
			if criteria.DistancePercentage < minPercentage || minPercentage == -1 {
				minPercentage = criteria.DistancePercentage
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

	var (
		shouldBranch bool
	)

	shouldBranch = i.checkShouldBranch(centroids)
	isAlreadyMax = !shouldBranch

	if !isAlreadyMax {
		optRange, isAlreadyMax = i.filterBlacklisted(optRange)
		if !isAlreadyMax {
			centroidsToNodes := map[s2.CellID][]string{}

			for _, node := range ordered {
				for _, cellId := range centroids {
					centroidsToNodes[cellId] = append(centroidsToNodes[cellId], node)
				}
			}

			exploredCentroids := map[s2.CellID]bool{}

			for _, cellId := range centroids {
				sort.Slice(centroidsToNodes[cellId], func(i, j int) bool {
					nodeI := centroidsToNodes[cellId][i]
					nodeJ := centroidsToNodes[cellId][j]

					distIToCell := sortingCriteria[nodeI].(sortingCriteriaType)[cellId].DistancePercentage
					distJToCell := sortingCriteria[nodeJ].(sortingCriteriaType)[cellId].DistancePercentage

					return distIToCell < distJToCell
				})
				_, exploredCentroids[cellId] = i.centroidsExtended[cellId]
			}

			actionArgs = make([]interface{}, ilArgsNum, ilArgsNum)
			actionArgs[ilActionTypeArgIndex] = actions.MultipleExtendDeploymentId
			actionArgs[ilCentroidDistancesToNodesIndex] = centroidsToNodes
			actionArgs[ilExploringCentroidsIndex] = exploredCentroids
		}
	}

	return
}

func (i *idealLatency) GenerateDomain(arg interface{}) (domain Domain, info map[string]interface{}, success bool) {
	value, ok := i.deployment.Exploring.Load(Myself.Id)
	if ok {
		exploringTTL := value.(exploringMapValue)
		log.Debugf("my exploringTTL is %d(%d)", exploringTTL, maxExploringTTL)
		if exploringTTL+1 == maxExploringTTL {
			return nil, nil, true
		}
	}

	value, ok = i.deployment.Environment.GetMetric(metrics.MetricLocationInVicinity)
	if !ok {
		log.Debugf("no value for metric %s", metrics.MetricLocationInVicinity)
		return nil, nil, false
	}

	locationsInVicinity := value.(map[string]interface{})

	centroids := arg.([]s2.CellID)

	var (
		myDists       = map[s2.CellID]float64{}
		centroidCells = map[s2.CellID]s2.Cell{}
	)
	for _, centroid := range centroids {
		centroidCell := s2.CellFromCellID(centroid)
		myDists[centroid] = utils.ChordAngleToKM(s2.CellFromCellID(i.myLocation).DistanceToCell(centroidCell))
		log.Debugf("mydist from %s to %s, %f", i.myLocation.ToToken(), centroid.ToToken(), myDists[centroid])
		centroidCells[centroid] = centroidCell
	}

	log.Debugf("nodes in vicinity: %+v", locationsInVicinity)
	info = map[string]interface{}{}
	for nodeId, cellValue := range locationsInVicinity {
		_, okC := i.deployment.Children.Load(nodeId)
		_, okS := i.deployment.Suspected.Load(nodeId)
		if okC || okS || nodeId == Myself.Id {
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
			delta := utils.ChordAngleToKM(s2.CellFromCellID(location).DistanceToCell(centroidCells[centroidId]))

			if nodeId == i.deployment.ParentId {
				nodeCentroidsMap = info[hiddenParentId].(sortingCriteriaType)
			} else {
				nodeCentroidsMap = info[nodeId].(sortingCriteriaType)
			}
			nodeCentroidsMap[centroidId] = &nodeWithDistance{
				NodeId:             nodeId,
				DistancePercentage: delta / myDists[centroidId],
			}
			log.Debugf("distance from %s(%s) to %s, %f", nodeId, location.ToToken(), centroidId.ToToken(), delta)
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

func (i *idealLatency) GenerateAction(targets []string, args ...interface{}) actions.Action {
	log.Debugf("generating action %s", (args[ilActionTypeArgIndex]).(string))

	switch args[ilActionTypeArgIndex].(string) {
	case actions.MultipleExtendDeploymentId:
		centroidsToNodes := args[ilCentroidDistancesToNodesIndex].(map[s2.CellID][]string)
		nodeCells := map[string][]s2.CellID{}
		var (
			nodesToExtendTo  []string
			targetsExploring = map[string]int{}
		)

		for cellId, nodesOrdered := range centroidsToNodes {
			var selectedNode string
			for _, node := range nodesOrdered {
				for _, target := range targets {
					if target == node {
						selectedNode = target
						break
					}
				}

				if selectedNode != "" {
					break
				}
			}

			if selectedNode == "" {
				panic(fmt.Sprintf("could not find a suitable node for cell %d, had %+v %+v", cellId,
					nodesOrdered, targets))
			}

			cells, ok := nodeCells[selectedNode]
			if !ok {
				cells = []s2.CellID{cellId}
				nodeCells[selectedNode] = cells
				nodesToExtendTo = append(nodesToExtendTo, selectedNode)
			} else {
				nodeCells[selectedNode] = append(nodeCells[selectedNode], cellId)
			}
		}

		exploredTTL := deployerAPI.NotExploringTTL
		value, ok := i.deployment.Exploring.Load(Myself.Id)
		if ok {
			exploredTTL = value.(exploringMapValue)
		}

		_, imExplored := i.deployment.Exploring.Load(Myself.Id)
		log.Debugf("im being explored %t", imExplored)
		for node, cells := range nodeCells {
			targetsExploring[node] = 0
			if imExplored {
				targetsExploring[node] = exploredTTL + 1
				continue
			}
			for _, cellId := range cells {
				_, centroidExtended := i.centroidsExtended[cellId]
				_, iAmExploring := i.deployment.Exploring.Load(Myself)
				if !centroidExtended && !iAmExploring {
					targetsExploring[node] = deployerAPI.NotExploringTTL
					break
				}
			}
		}

		toExclude := map[string]interface{}{}
		i.deployment.Blacklist.Range(func(key, value interface{}) bool {
			nodeId := key.(string)
			toExclude[nodeId] = nil
			return true
		})
		i.deployment.Exploring.Range(func(key, value interface{}) bool {
			nodeId := key.(string)
			toExclude[nodeId] = nil
			return true
		})

		return actions.NewMultipleExtendDeploymentAction(i.deployment.DeploymentId, nodesToExtendTo, nodeCells,
			targetsExploring, i.extendedCentroidCallback, toExclude, i.deployment.setNodeAsExploring)
	}

	return nil
}

func (i *idealLatency) extendedCentroidCallback(centroid s2.CellID) {
	i.centroidsExtended[centroid] = nil
}

func (i *idealLatency) GetDependencies() (metrics []string) {
	return i.dependencies
}

func (i *idealLatency) calcFurthestChildDistance(avgLocation s2.CellID) (furthestChild string,
	furthestChildDistance float64) {
	furthestChildDistance = -1.0

	i.deployment.Children.Range(func(key, value interface{}) bool {
		childId := key.(deploymentChildrenMapKey)
		child := value.(deploymentChildrenMapValue)
		delta := utils.ChordAngleToKM(s2.CellFromCellID(child.Location).DistanceToCell(s2.CellFromCellID(avgLocation)))

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

		furthestChildDistance = utils.ChordAngleToKM(s2.CellFromCellID(location).
			DistanceToCell(s2.CellFromCellID(avgLocation)))
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

func (i *idealLatency) checkShouldBranch(centroids []s2.CellID) bool {
	numChildren := 0
	i.deployment.Children.Range(func(key, value interface{}) bool {
		numChildren++
		return true
	})

	centroidDistSum := 0.
	for _, centroid := range centroids {
		centroidDistSum += utils.ChordAngleToKM(s2.CellFromCellID(i.myLocation).DistanceToCell(s2.CellFromCellID(centroid)))
	}
	avgDistanceToCentroids := centroidDistSum / float64(len(centroids))

	distanceFactor := maxDistance / (maxDistance - (avgDistanceToCentroids - satisfiedDistance))
	childrenFactor := (((maxChildren + 1.) / (float64(numChildren) + 1.)) - 1.) / maxChildren
	cosDelta := 0.
	sinDelta := 0.
	for _, centroid := range centroids {
		latDelta := centroid.LatLng().Lat.Degrees() - i.myLocation.LatLng().Lat.Degrees()
		lngDelta := centroid.LatLng().Lng.Degrees() - i.myLocation.LatLng().Lng.Degrees()
		angle := math.Atan2(lngDelta, latDelta)
		cosDelta += math.Cos(angle)
		sinDelta += math.Sin(angle)
	}
	accumulatedDiff := cosDelta + sinDelta
	heterogeneity := (2. / (accumulatedDiff + 1.)) - 1.

	heterogeneityFactor := float64(numChildren) + 2. - (heterogeneity * float64(numChildren))
	branchingFactor := childrenFactor * distanceFactor * heterogeneityFactor
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
