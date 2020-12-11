package autonomic

import (
	"sort"
	"sync"
	"time"

	autonomicAPI "github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
	deployerAPI "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	autonomicUtils "github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/servers"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/scheduler"
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

		archFactory  archimedes.ClientFactory
		autoFactory  autonomic.ClientFactory
		deplFactory  deployer.ClientFactory
		schedFactory scheduler.ClientFactory
	}
)

const (
	initialDeploymentDelay = 20 * time.Second
)

func newSystem(autoFactory autonomic.ClientFactory, archFactory archimedes.ClientFactory,
	deplFactory deployer.ClientFactory, schedFactory scheduler.ClientFactory) *system {
	return &system{
		deployments:      &sync.Map{},
		exitChans:        &sync.Map{},
		env:              environment.NewEnvironment(),
		suspected:        &sync.Map{},
		deployerClient:   deplFactory.New(servers.DeployerLocalHostPort),
		archimedesClient: archFactory.New(servers.ArchimedesLocalHostPort),
		archFactory:      archFactory,
		autoFactory:      autoFactory,
		deplFactory:      deplFactory,
		schedFactory:     schedFactory,
	}
}

func (a *system) addDeployment(deploymentID, strategyID string, depthFactor float64, exploringTTL int) {
	if value, ok := a.deployments.Load(deploymentID); ok {
		depl := value.(deploymentsMapValue)

		if exploringTTL != deployerAPI.NotExploringTTL {
			depl.Exploring.Store(deployment.Myself.ID, exploringTTL)
		} else {
			depl.Exploring.Delete(deployment.Myself.ID)
		}

		exitChan := make(chan interface{})
		a.exitChans.Store(deploymentID, exitChan)

		go a.handleDeployment(depl, exitChan)
	}

	log.Debugf("new deployment %s has exploringTTL %d", deploymentID, exploringTTL)

	s, err := deployment.New(deploymentID, strategyID, a.suspected, depthFactor, a.env, a.autoFactory, a.archFactory,
		a.deplFactory, a.schedFactory)
	if err != nil {
		panic(err)
	}

	if exploringTTL != deployerAPI.NotExploringTTL {
		s.Exploring.Store(deployment.Myself.ID, exploringTTL)
	}

	a.deployments.Store(deploymentID, s)

	exitChan := make(chan interface{})
	a.exitChans.Store(deploymentID, exitChan)

	go a.handleDeployment(s, exitChan)
}

func (a *system) removeDeployment(deploymentID string) {
	_, ok := a.deployments.Load(deploymentID)
	if !ok {
		return
	}

	value, ok := a.exitChans.Load(deploymentID)
	if !ok {
		return
	}

	exitChan := value.(chan interface{})
	close(exitChan)

	a.deployments.Delete(deploymentID)
}

func (a *system) addDeploymentChild(deploymentID string, child *utils.Node) {
	value, ok := a.deployments.Load(deploymentID)
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

	cellValue, ok := vicinityMetric.Locations[child.ID]
	if !ok {
		log.Errorf("no location for child %s", child.ID)

		return
	}

	log.Debugf("adding child %s", child.ID)

	location := s2.CellIDFromToken(cellValue)

	a.suspected.Delete(child.ID)
	s.AddChild(child, location)
}

func (a *system) removeDeploymentChild(deploymentID, childID string) {
	log.Debugf("removing child %s for deployment %s", childID, deploymentID)

	value, ok := a.deployments.Load(deploymentID)
	if !ok {
		return
	}

	log.Debug("removed")

	s := value.(deploymentsMapValue)
	s.AddSuspectedChild(childID)
	s.RemoveChild(childID)
}

func (a *system) setDeploymentParent(deploymentID string, parent *utils.Node) {
	value, ok := a.deployments.Load(deploymentID)
	if !ok {
		return
	}

	s := value.(deploymentsMapValue)
	s.SetParent(parent)
}

func (a *system) getDeployments() (deployments map[string]*deployment.Deployment) {
	deployments = map[string]*deployment.Deployment{}

	a.deployments.Range(func(key, value interface{}) bool {
		deploymentID := key.(deploymentsMapKey)
		s := value.(deploymentsMapValue)

		deployments[deploymentID] = s

		return true
	})

	return
}

func (a *system) isNodeInVicinity(nodeID string) bool {
	vicinity := a.getVicinity()
	_, ok := vicinity.Nodes[nodeID]

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

	ordered := make([]*utils.Node, 0, len(vicinity.Nodes))

	for nodeID, node := range vicinity.Nodes {
		if _, ok = toExclude[nodeID]; ok {
			continue
		}

		ordered = append(ordered, node)
	}

	locationCells := make([]s2.Cell, len(locations))

	for _, location := range locations {
		locationCells = append(locationCells, s2.CellFromCellID(location))
	}

	sort.Slice(ordered, func(i, j int) bool {
		iID := s2.CellIDFromToken(vicinity.Locations[ordered[i].ID])
		jID := s2.CellIDFromToken(vicinity.Locations[ordered[j].ID])

		iCell := s2.CellFromCellID(iID)
		jCell := s2.CellFromCellID(jID)

		iDistSum := 0.
		jDistSum := 0.
		for _, locationCell := range locationCells {
			iDistSum += servers.ChordAngleToKM(iCell.DistanceToCell(locationCell))
			jDistSum += servers.ChordAngleToKM(jCell.DistanceToCell(locationCell))
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
	for nodeID, cellToken := range vicinityMetric.Locations {
		vicinity.Locations[nodeID] = s2.CellIDFromToken(cellToken)
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

		log.Debugf("evaluating deployment %s", deployment.DeploymentID)

		action := deployment.GenerateAction()
		if action != nil {
			log.Debugf("generated action of type %s for deployment %s", action.GetActionID(), deployment.DeploymentID)
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
		log.Errorf("could not execute action of type %s", action.GetActionID())
	}
}

func (a *system) getLoad(deploymentID string) (float64, bool) {
	value, ok := a.deployments.Load(deploymentID)
	if !ok {
		return 0, false
	}

	return value.(deploymentsMapValue).GetLoad(), true
}

func (a *system) setExploreSuccess(deploymentID, childID string) bool {
	value, ok := a.deployments.Load(deploymentID)
	if !ok {
		return false
	}

	return value.(deploymentsMapValue).SetExploreSuccess(childID)
}
