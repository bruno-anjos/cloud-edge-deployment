package deployment

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"

	deployerAPI "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/servers"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/environment"
	"github.com/golang/geo/s2"
	log "github.com/sirupsen/logrus"
)

const (
	maximumDistancePercentage = 1.2
	satisfiedDistance         = 20.
	maxDistance               = servers.EarthRadius * math.Pi
	maxChildren               = 4.
	branchingCutoff           = 1

	maxExploringTTL    = 0
	nonMaxedPercentage = 1.

	idealLatencyGoalID = "GOAL_IDEAL_LATENCY"

	hiddenParentID = "_parent"
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

	centroidToNodesType = map[s2.CellID][]*utils.Node

	nodeWithDistance struct {
		NodeID             string
		DistancePercentage float64
	}

	sortingCriteriaType = map[s2.CellID]*nodeWithDistance
)

type idealLatency struct {
	deployment        *Deployment
	myLocation        s2.CellID
	centroidsExtended map[s2.CellID]interface{}
	depthFactor       float64
	deplLogger        *log.Entry
}

func newIdealLatencyGoal(deployment *Deployment) *idealLatency {
	deplLogger := log.WithFields(log.Fields{"DEPL": deployment.DeploymentID, "GOAL": "IDEAL_LATENCY"})

	locationToken, ok := os.LookupEnv(utils.LocationEnvVarName)
	if !ok {
		deplLogger.Panic("could not get location from environment")
	}

	return &idealLatency{
		deployment:        deployment,
		myLocation:        s2.CellIDFromToken(locationToken),
		centroidsExtended: map[s2.CellID]interface{}{},
		depthFactor:       deployment.DepthFactor,
		deplLogger:        deplLogger,
	}
}

func (i *idealLatency) Optimize(optDomain domain) (isAlreadyMax bool, optRange result, actionArgs []interface{}) {
	isAlreadyMax = true
	optRange = optDomain
	actionArgs = nil

	centroids := environment.GetClientCentroids(i.deployment.Environment.DemmonCli, i.deployment.DeploymentID, Myself)
	if len(centroids) == 0 {
		return
	}

	centroidTokens := make([]string, len(centroids))

	for idx, centroid := range centroids {
		centroidTokens[idx] = centroid.ToToken()
	}

	log.Debugf("got centroids %+v", centroidTokens)
	centroidTokens = nil

	candidates, sortingCriteria, ok := i.GenerateDomain(centroids)
	if !ok {
		return
	}

	candidateIds := make([]string, len(candidates))

	for idx, candidate := range candidates {
		candidateIds[idx] = candidate.ID
	}

	i.deplLogger.Debugf("%s generated domain %+v", idealLatencyGoalID, candidateIds)
	filtered := i.Filter(candidates, optDomain)

	nodeMinDistances := map[string]interface{}{}

	for _, node := range filtered {
		minPercentage := -1.
		for _, criteria := range sortingCriteria[node.ID].(sortingCriteriaType) {
			if criteria.DistancePercentage < minPercentage || minPercentage == -1 {
				minPercentage = criteria.DistancePercentage
			}
		}

		nodeMinDistances[node.ID] = minPercentage
	}

	filteredIds := make([]string, len(filtered))
	for idx, filter := range filtered {
		filteredIds[idx] = filter.ID
	}

	i.deplLogger.Debugf("%s filtered result %+v", idealLatencyGoalID, filteredIds)

	ordered := i.Order(filtered, nodeMinDistances)
	orderedIds := make([]string, len(ordered))

	for idx, order := range ordered {
		orderedIds[idx] = order.ID
	}

	i.deplLogger.Debugf("%s ordered result %+v", idealLatencyGoalID, orderedIds)

	optRange, isAlreadyMax = i.Cutoff(ordered, nodeMinDistances)

	optRangeIds := make([]string, len(optRange))
	for idx, rangeNode := range optRange {
		optRangeIds[idx] = rangeNode.ID
	}

	i.deplLogger.Debugf("%s cutoff result (%t) %+v", idealLatencyGoalID, isAlreadyMax, optRangeIds)

	if len(optRange) == 0 {
		return
	}

	shouldBranch := i.checkShouldBranch(centroids)
	isAlreadyMax = !shouldBranch

	if !isAlreadyMax {
		optRange, isAlreadyMax = i.filterBlacklisted(optRange)
		if !isAlreadyMax {
			centroidsToNodes := centroidToNodesType{}

			for _, node := range ordered {
				for _, cellID := range centroids {
					centroidsToNodes[cellID] = append(centroidsToNodes[cellID], node)
				}
			}

			exploredCentroids := map[s2.CellID]bool{}

			for _, cellID := range centroids {
				cellAux := cellID
				sort.Slice(centroidsToNodes[cellAux], func(i, j int) bool {
					nodeI := centroidsToNodes[cellAux][i]
					nodeJ := centroidsToNodes[cellAux][j]

					distIToCell := sortingCriteria[nodeI.ID].(sortingCriteriaType)[cellAux].DistancePercentage
					distJToCell := sortingCriteria[nodeJ.ID].(sortingCriteriaType)[cellAux].DistancePercentage

					return distIToCell < distJToCell
				})

				_, exploredCentroids[cellID] = i.centroidsExtended[cellID]
			}

			actionArgs = make([]interface{}, ilArgsNum)
			actionArgs[ilActionTypeArgIndex] = actions.MultipleExtendDeploymentID
			actionArgs[ilCentroidDistancesToNodesIndex] = centroidsToNodes
			actionArgs[ilExploringCentroidsIndex] = exploredCentroids
		}
	}

	return isAlreadyMax, optRange, actionArgs
}

func (i *idealLatency) GenerateDomain(arg interface{}) (domain domain, info map[string]interface{}, success bool) {
	domain = nil
	info = nil
	success = false

	value, ok := i.deployment.Exploring.Load(Myself.ID)
	if ok {
		exploringTTL := value.(exploringMapValue)
		i.deplLogger.Debugf("my exploringTTL is %d(%d)", exploringTTL, maxExploringTTL)

		if exploringTTL+1 >= maxExploringTTL {
			return nil, nil, true
		}
	}

	vicinity := i.deployment.Environment.GetVicinity()
	locations := i.deployment.Environment.GetLocationInVicinity()

	centroids := arg.([]s2.CellID)

	var (
		myDists       = map[s2.CellID]float64{}
		centroidCells = map[s2.CellID]s2.Cell{}
	)

	for _, centroid := range centroids {
		centroidCell := s2.CellFromCellID(centroid)
		myDists[centroid] = servers.ChordAngleToKM(s2.CellFromCellID(i.myLocation).DistanceToCell(centroidCell))
		i.deplLogger.Debugf("my dist from %s to %s, %f", i.myLocation.ToToken(), centroid.ToToken(),
			myDists[centroid])

		centroidCells[centroid] = centroidCell
	}

	i.deplLogger.Debugf("nodes in vicinity: %+v", vicinity)

	info = map[string]interface{}{}

	for nodeID, node := range vicinity {
		_, okC := i.deployment.Children.Load(nodeID)
		_, okS := i.deployment.Suspected.Load(nodeID)

		location, okL := locations[node.ID]

		if okC || okS || nodeID == Myself.ID || !okL {
			i.deplLogger.Debugf("ignoring %s", nodeID)

			continue
		}

		// create node map for centroids and respective distances
		if i.deployment.Parent != nil && nodeID == i.deployment.Parent.ID {
			info[hiddenParentID] = sortingCriteriaType{}
		} else {
			info[nodeID] = sortingCriteriaType{}
			domain = append(domain, node)
		}

		var nodeCentroidsMap sortingCriteriaType

		for _, centroidID := range centroids {
			delta := servers.ChordAngleToKM(s2.CellFromCellID(location).DistanceToCell(centroidCells[centroidID]))

			if i.deployment.Parent != nil && nodeID == i.deployment.Parent.ID {
				nodeCentroidsMap = info[hiddenParentID].(sortingCriteriaType)
			} else {
				nodeCentroidsMap = info[nodeID].(sortingCriteriaType)
			}

			var percentage float64

			if myDists[centroidID] == 0 {
				if delta == 0 {
					percentage = 1
				} else {
					percentage = math.Inf(1)
				}
			} else {
				percentage = delta / myDists[centroidID]
			}

			if myDists[centroidID] != 0. {
				percentage = delta / myDists[centroidID]
			} else if myDists[centroidID] == 0 && delta == 0 {
			}

			nodeCentroidsMap[centroidID] = &nodeWithDistance{
				NodeID:             nodeID,
				DistancePercentage: percentage,
			}
		}
	}

	success = true

	return domain, info, success
}

func (i *idealLatency) Order(candidates domain, sortingCriteria map[string]interface{}) (ordered result) {
	ordered = candidates
	sort.Slice(ordered, func(i, j int) bool {
		return sortingCriteria[ordered[i].ID].(float64) < sortingCriteria[ordered[j].ID].(float64)
	})

	return
}

func (i *idealLatency) Filter(candidates, domain domain) (filtered result) {
	return defaultFilter(candidates, domain)
}

func (i *idealLatency) Cutoff(candidates domain, candidatesCriteria map[string]interface{}) (cutoff result,
	maxed bool) {
	maxed = true

	candidateClient := i.deployment.deplFactory.New()

	for _, candidate := range candidates {
		percentage := candidatesCriteria[candidate.ID].(float64)
		i.deplLogger.Debugf("candidate %s distance percentage (me) %f", candidate, percentage)

		if percentage < maximumDistancePercentage {
			addr := candidate.Addr + ":" + strconv.Itoa(deployer.Port)

			has, _ := candidateClient.HasDeployment(addr, i.deployment.DeploymentID)
			if has {
				i.deplLogger.Debugf("candidate %s already has deployment %s", candidate, i.deployment.DeploymentID)

				continue
			}

			cutoff = append(cutoff, candidate)
		}

		if percentage < nonMaxedPercentage {
			maxed = false
		}
	}

	return
}

func (i *idealLatency) GenerateAction(targets result, args ...interface{}) actions.Action {
	i.deplLogger.Debugf("generating action %s", (args[ilActionTypeArgIndex]).(string))

	if args[ilActionTypeArgIndex].(string) == actions.MultipleExtendDeploymentID {
		return i.generateMultipleExtendAction(targets, args...)
	}

	return nil
}

func (i *idealLatency) generateMultipleExtendAction(targets result, args ...interface{}) actions.Action {
	centroidsToNodes := args[ilCentroidDistancesToNodesIndex].(centroidToNodesType)
	nodeCells := map[string][]s2.CellID{}

	var (
		nodesToExtendTo  []*utils.Node
		targetsExploring = map[string]int{}
	)

	for cellID, nodesOrdered := range centroidsToNodes {
		var selectedNode *utils.Node

		for _, node := range nodesOrdered {
			for _, target := range targets {
				if target.ID == node.ID {
					selectedNode = node

					break
				}
			}

			if selectedNode != nil {
				break
			}
		}

		if selectedNode == nil {
			panic(fmt.Sprintf("could not find a suitable node for cell %d, had %+v %+v", cellID,
				nodesOrdered, targets))
		}

		_, ok := nodeCells[selectedNode.ID]
		if !ok {
			cells := []s2.CellID{cellID}
			nodeCells[selectedNode.ID] = cells

			nodesToExtendTo = append(nodesToExtendTo, selectedNode)
		} else {
			nodeCells[selectedNode.ID] = append(nodeCells[selectedNode.ID], cellID)
		}
	}

	exploredTTL := deployerAPI.NotExploringTTL

	value, imExplored := i.deployment.Exploring.Load(Myself.ID)
	if imExplored {
		exploredTTL = value.(exploringMapValue)
	}

	i.deplLogger.Debugf("im being explored %t", imExplored)

	for node, cells := range nodeCells {
		targetsExploring[node] = 0
		if imExplored {
			targetsExploring[node] = exploredTTL + 1

			continue
		}

		for _, cellID := range cells {
			_, centroidExtended := i.centroidsExtended[cellID]

			if !centroidExtended && !imExplored {
				targetsExploring[node] = deployerAPI.NotExploringTTL

				break
			}
		}
	}

	toExclude := map[string]interface{}{}

	i.deployment.Blacklist.Range(func(key, value interface{}) bool {
		nodeID := key.(string)
		toExclude[nodeID] = nil

		return true
	})
	i.deployment.Exploring.Range(func(key, value interface{}) bool {
		nodeID := key.(string)
		toExclude[nodeID] = nil

		return true
	})

	return actions.NewMultipleExtendDeploymentAction(i.deployment.DeploymentID, nodesToExtendTo, nodeCells,
		targetsExploring, i.extendedCentroidCallback, toExclude, i.deployment.setNodeAsExploring,
		i.deployment.deplFactory)
}

func (i *idealLatency) extendedCentroidCallback(centroid s2.CellID) {
	i.centroidsExtended[centroid] = nil
}

func (i *idealLatency) calcFurthestChildDistance(avgLocation s2.CellID) (furthestChild string,
	furthestChildDistance float64) {
	furthestChildDistance = -1.0

	i.deployment.Children.Range(func(key, value interface{}) bool {
		childID := key.(deploymentChildrenMapKey)
		child := value.(deploymentChildrenMapValue)
		delta := servers.ChordAngleToKM(s2.CellFromCellID(child.Location).DistanceToCell(s2.CellFromCellID(
			avgLocation)))

		if delta > furthestChildDistance {
			furthestChildDistance = delta
			furthestChild = childID
		}

		i.deplLogger.Debugf("child %s", childID)

		return true
	})

	if furthestChildDistance == -1.0 {
		furthestChildDistance = servers.ChordAngleToKM(s2.CellFromCellID(i.myLocation).
			DistanceToCell(s2.CellFromCellID(avgLocation)))
	}

	return furthestChild, furthestChildDistance
}

func (i *idealLatency) GetID() string {
	return idealLatencyGoalID
}

// func (i *idealLatency) checkProcessingTime() bool {
// 	processintTimeMetric := environment.GetProcessingTimePerDeploymentMetricID(i.deployment.DeploymentID)
//
// 	value, ok := i.deployment.Environment.GetMetric(processintTimeMetric)
// 	if !ok {
// 		i.deplLogger.Debugf("no value for metric %s", processintTimeMetric)
//
// 		return false
// 	}
//
// 	processingTime := value.(float64)
// 	clientLatencyMetric := environment.GetClientLatencyPerDeploymentMetricID(i.deployment.DeploymentID)
//
// 	value, ok = i.deployment.Environment.GetMetric(clientLatencyMetric)
// 	if !ok {
// 		i.deplLogger.Debugf("no value for metric %s", clientLatencyMetric)
//
// 		return false
// 	}
//
// 	latency := value.(float64)
//
// 	processingTimePart := float32(processingTime) / float32(latency)
// 	if processingTimePart > processingThreshold {
// 		i.deplLogger.Debugf("most of the client latency is due to processing time (%f)", processingTimePart)
//
// 		return true
// 	}
//
// 	return false
// }

func (i *idealLatency) checkShouldBranch(centroids []s2.CellID) bool {
	numChildren := 0

	i.deployment.Children.Range(func(key, value interface{}) bool {
		numChildren++

		return true
	})

	centroidDistSum := 0.
	for _, centroid := range centroids {
		centroidDistSum += servers.ChordAngleToKM(s2.CellFromCellID(i.myLocation).DistanceToCell(s2.CellFromCellID(
			centroid)))
	}

	avgDistanceToCentroids := centroidDistSum / float64(len(centroids))

	distanceFactor := maxDistance / (maxDistance - (avgDistanceToCentroids - satisfiedDistance))
	childrenFactor := (((maxChildren + 1.) / (float64(numChildren) + 1.)) - 1.) / maxChildren
	// cosDelta := 0.
	// sinDelta := 0.
	//
	// for _, centroid := range centroids {
	// 	latDelta := centroid.LatLng().Lat.Degrees() - i.myLocation.LatLng().Lat.Degrees()
	// 	lngDelta := centroid.LatLng().Lng.Degrees() - i.myLocation.LatLng().Lng.Degrees()
	// 	angle := math.Atan2(lngDelta, latDelta)
	// 	cosDelta += math.Cos(angle)
	// 	sinDelta += math.Sin(angle)
	// }
	//
	// accumulatedDiff := cosDelta + sinDelta
	//
	// var heterogeneityFactor float64
	// if numChildren < 1 {
	// 	heterogeneityFactor = 1
	// } else {
	// 	heterogeneityFactor = accumulatedDiff / float64(numChildren) //nolint:gomnd
	// }

	branchingFactor := childrenFactor * distanceFactor * i.depthFactor
	i.deplLogger.Debugf("branching factor %f (%d; %f * %f * %f)", branchingFactor, numChildren, childrenFactor,
		distanceFactor, i.depthFactor)

	validBranch := branchingFactor > branchingCutoff
	i.deplLogger.Debugf("should branch: %t", validBranch)

	return validBranch
}

func (i *idealLatency) filterBlacklisted(original result) (result, bool) {
	var newRange result

	for _, node := range original {
		if _, ok := i.deployment.Blacklist.Load(node); !ok {
			newRange = append(newRange, node)
		}
	}

	i.deplLogger.Debugf("after filtering blacklisted: %+v", newRange)

	return newRange, len(newRange) == 0
}
