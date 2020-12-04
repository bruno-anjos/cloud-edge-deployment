package autonomic

import (
	"sort"
	"sync"
	"time"

	autonomicAPI "github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
	deployerAPI "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	autonomicUtils "github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/utils"
	internalUtils "github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/deployment"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/environment"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/metrics"
	"github.com/golang/geo/s2"
	log "github.com/sirupsen/logrus"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
)

type (
	deploymentsMapKey   = string
	deploymentsMapValue = *deployment.Deployment
)

type (
	system struct {
		deployments *sync.Map
		exitChans   *sync.Map
		env         *environment.Environment
		suspected   *sync.Map

		deployerClient   deployer.Client
		archimedesClient archimedes.Client

		deplFactory deployer.ClientFactory
		autoFactory autonomic.ClientFactory
	}
)

const (
	initialDeploymentDelay = 20 * time.Second
)

func newSystem(deplFactory deployer.ClientFactory, archFactory archimedes.ClientFactory,
	autoFactory autonomic.ClientFactory) *system {
	return &system{
		deployments:      &sync.Map{},
		exitChans:        &sync.Map{},
		env:              environment.NewEnvironment(),
		suspected:        &sync.Map{},
		deployerClient:   deplFactory.New(internalUtils.DeployerLocalHostPort),
		archimedesClient: archFactory.New(internalUtils.ArchimedesLocalHostPort),
		deplFactory:      deplFactory,
		autoFactory:      autoFactory,
	}
}

func (a *system) addDeployment(deploymentId, strategyId string, depthFactor float64, exploringTTL int) {
	if value, ok := a.deployments.Load(deploymentId); ok {
		depl := value.(deploymentsMapValue)
		if exploringTTL != deployerAPI.NotExploringTTL {
			depl.Exploring.Store(deployment.Myself.Id, exploringTTL)
		} else {
			depl.Exploring.Delete(deployment.Myself.Id)
		}
		exitChan := make(chan interface{})
		a.exitChans.Store(deploymentId, exitChan)
		go a.handleDeployment(depl, exitChan)
	}

	log.Debugf("new deployment %s has exploringTTL %d", deploymentId, exploringTTL)

	s, err := deployment.New(deploymentId, strategyId, a.suspected, depthFactor, a.env, a.autoFactory)
	if err != nil {
		panic(err)
	}

	if exploringTTL != deployerAPI.NotExploringTTL {
		s.Exploring.Store(deployment.Myself.Id, exploringTTL)
	}

	a.deployments.Store(deploymentId, s)
	exitChan := make(chan interface{})
	a.exitChans.Store(deploymentId, exitChan)
	go a.handleDeployment(s, exitChan)

	return
}

func (a *system) removeDeployment(deploymentId string) {
	_, ok := a.deployments.Load(deploymentId)
	if !ok {
		return
	}

	value, ok := a.exitChans.Load(deploymentId)
	if !ok {
		return
	}

	exitChan := value.(chan interface{})
	close(exitChan)

	a.deployments.Delete(deploymentId)
}

func (a *system) addDeploymentChild(deploymentId string, child *utils.Node) {
	value, ok := a.deployments.Load(deploymentId)
	if !ok {
		return
	}

	s := value.(deploymentsMapValue)
	value, ok = a.env.GetMetric(metrics.MetricLocationInVicinity)
	if !ok {
		log.Errorf("no metric %s", metrics.MetricLocationInVicinity)
		return
	}

	var vicinityMetric metrics.VicinityMetric
	err := mapstructure.Decode(value, &vicinityMetric)
	if err != nil {
		panic(err)
	}

	cellValue, ok := vicinityMetric.Locations[child.Id]
	if !ok {
		log.Errorf("no location for child %s", child.Id)
		return
	}

	log.Debugf("adding child %s", child.Id)

	location := s2.CellIDFromToken(cellValue)

	a.suspected.Delete(child.Id)
	s.AddChild(child, location)
}

func (a *system) removeDeploymentChild(deploymentId, childId string) {
	log.Debugf("removing child %s for deployment %s", childId, deploymentId)

	value, ok := a.deployments.Load(deploymentId)
	if !ok {
		return
	}

	log.Debug("removed")

	s := value.(deploymentsMapValue)
	s.AddSuspectedChild(childId)
	s.RemoveChild(childId)
}

func (a *system) setDeploymentParent(deploymentId string, parent *utils.Node) {
	value, ok := a.deployments.Load(deploymentId)
	if !ok {
		return
	}

	s := value.(deploymentsMapValue)
	s.SetParent(parent)
}

func (a *system) getDeployments() (deployments map[string]*deployment.Deployment) {
	deployments = map[string]*deployment.Deployment{}

	a.deployments.Range(func(key, value interface{}) bool {
		deploymentId := key.(deploymentsMapKey)
		s := value.(deploymentsMapValue)

		deployments[deploymentId] = s

		return true
	})

	return
}

func (a *system) isNodeInVicinity(nodeId string) bool {
	vicinity := a.getVicinity()
	_, ok := vicinity.Nodes[nodeId]

	return ok
}

func (a *system) closestNodeTo(locations []s2.CellID, toExclude map[string]interface{}) *utils.Node {
	value, ok := a.env.GetMetric(metrics.MetricLocationInVicinity)
	if !ok {
		return nil
	}

	var vicinity metrics.VicinityMetric
	err := mapstructure.Decode(value, &vicinity)
	if err != nil {
		panic(err)
	}
	var ordered []*utils.Node

	for nodeId, node := range vicinity.Nodes {
		if _, ok = toExclude[nodeId]; ok {
			continue
		}
		ordered = append(ordered, node)
	}

	var locationCells []s2.Cell
	for _, location := range locations {
		locationCells = append(locationCells, s2.CellFromCellID(location))
	}

	sort.Slice(ordered, func(i, j int) bool {
		iId := s2.CellIDFromToken(vicinity.Locations[ordered[i].Id])
		jId := s2.CellIDFromToken(vicinity.Locations[ordered[j].Id])

		iCell := s2.CellFromCellID(iId)
		jCell := s2.CellFromCellID(jId)

		iDistSum := 0.
		jDistSum := 0.
		for _, locationCell := range locationCells {
			iDistSum += internalUtils.ChordAngleToKM(iCell.DistanceToCell(locationCell))
			jDistSum += internalUtils.ChordAngleToKM(jCell.DistanceToCell(locationCell))
		}

		return iDistSum < jDistSum
	})

	if len(ordered) < 1 {
		return nil
	}

	return ordered[0]
}

func (a *system) getVicinity() *autonomicAPI.Vicinity {
	value, ok := a.env.GetMetric(metrics.MetricLocationInVicinity)
	if !ok {
		return nil
	}

	var vicinityMetric metrics.VicinityMetric
	err := mapstructure.Decode(value, &vicinityMetric)
	if err != nil {
		panic(err)
	}

	vicinity := &autonomicAPI.Vicinity{
		Nodes:     vicinityMetric.Nodes,
		Locations: map[string]s2.CellID{},
	}
	for nodeId, cellToken := range vicinityMetric.Locations {
		vicinity.Locations[nodeId] = s2.CellIDFromToken(cellToken)
	}

	return vicinity
}

func (a *system) getMyLocation() (s2.CellID, error) {
	value, ok := a.env.GetMetric(metrics.MetricLocation)
	if !ok {
		return 0, errors.New("could not fetch my location")
	}

	return s2.CellIDFromToken(value.(string)), nil
}

func (a *system) handleDeployment(deployment *deployment.Deployment, exit <-chan interface{}) {
	time.Sleep(initialDeploymentDelay)

	timer := time.NewTimer(autonomicUtils.DefaultGoalCycleTimeout)

	for {
		select {
		case <-exit:
			return
		case <-timer.C:
		}

		log.Debugf("evaluating deployment %s", deployment.DeploymentId)

		action := deployment.GenerateAction()
		if action != nil {
			log.Debugf("generated action of type %s for deployment %s", action.GetActionId(), deployment.DeploymentId)
			a.performAction(action)
		}

		timer.Reset(autonomicUtils.DefaultGoalCycleTimeout)
	}
}

func (a *system) performAction(action actions.Action) {
	switch assertedAction := action.(type) {
	case *actions.RedirectAction:
		assertedAction.Execute(a.archimedesClient)
	case *actions.ExtendDeploymentAction:
		assertedAction.Execute(a.deployerClient)
	case *actions.MultipleExtendDeploymentAction:
		assertedAction.Execute(a.deployerClient)
	case *actions.RemoveDeploymentAction:
		assertedAction.Execute(a.deployerClient)
	default:
		log.Errorf("could not execute action of type %s", action.GetActionId())
	}
}

func (a *system) getLoad(deploymentId string) (float64, bool) {
	value, ok := a.deployments.Load(deploymentId)
	if !ok {
		return 0, false
	}

	return value.(deploymentsMapValue).GetLoad(), true
}

func (a *system) setExploreSuccess(deploymentId, childId string) bool {
	value, ok := a.deployments.Load(deploymentId)
	if !ok {
		return false
	}

	return value.(deploymentsMapValue).SetExploreSuccess(childId)
}
