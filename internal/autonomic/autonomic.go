package autonomic

import (
	"sort"
	"sync"
	"time"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/pkg/errors"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/deployment"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/environment"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/metrics"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/golang/geo/s2"
	log "github.com/sirupsen/logrus"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
)

type (
	deploymentsMapKey   = string
	deploymentsMapValue = *deployment.Deployment
)

const (
	defaultInterval        = 30 * time.Second
	initialDeploymentDelay = 30 * time.Second
	blacklistInterval      = (4 * 30) * time.Second
)

type (
	system struct {
		deployments *sync.Map
		env         *environment.Environment
		suspected   *sync.Map

		deployerClient   *deployer.Client
		archimedesClient *archimedes.Client
		exploring        sync.Map
	}

	exploringMapValue = chan interface{}
)

func newSystem() *system {
	return &system{
		deployments:      &sync.Map{},
		env:              environment.NewEnvironment(),
		suspected:        &sync.Map{},
		deployerClient:   deployer.NewDeployerClient(deployer.DefaultHostPort),
		archimedesClient: archimedes.NewArchimedesClient(archimedes.DefaultHostPort),
		exploring:        sync.Map{},
	}
}

func (a *system) addDeployment(deploymentId, strategyId string) {
	if _, ok := a.deployments.Load(deploymentId); ok {
		return
	}

	s, err := deployment.New(deploymentId, strategyId, a.suspected, a.env)
	if err != nil {
		panic(err)
	}

	a.deployments.Store(deploymentId, s)
	go a.handleDeployments(s)

	return
}

func (a *system) removeDeployment(deploymentId string) {
	a.deployments.Delete(deploymentId)
}

func (a *system) addDeploymentChild(deploymentId, childId string) {
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

	locations := value.(map[string]interface{})
	cellValue, ok := locations[childId]
	if !ok {
		log.Errorf("no location for child %s", childId)
		return
	}

	location := s2.CellIDFromToken(cellValue.(string))

	a.suspected.Delete(childId)
	s.AddChild(childId, location)
}

func (a *system) removeDeploymentChild(deploymentId, childId string) {
	value, ok := a.deployments.Load(deploymentId)
	if !ok {
		return
	}

	s := value.(deploymentsMapValue)
	s.AddSuspectedChild(childId)
	s.RemoveChild(childId)
}

func (a *system) setDeploymentParent(deploymentId, parentId string) {
	value, ok := a.deployments.Load(deploymentId)
	if !ok {
		return
	}

	s := value.(deploymentsMapValue)
	s.SetParent(parentId)
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
	_, ok := vicinity[nodeId]

	return ok
}

func (a *system) closestNodeTo(locations []s2.CellID, toExclude map[string]interface{}) (nodeId string) {
	value, ok := a.env.GetMetric(metrics.MetricLocationInVicinity)
	if !ok {
		return ""
	}

	vicinity := value.(map[string]interface{})
	var ordered []string

	for node := range vicinity {
		if _, ok = toExclude[node]; ok {
			continue
		}
		ordered = append(ordered, node)
	}

	var locationCells []s2.Cell
	for _, location := range locations {
		locationCells = append(locationCells, s2.CellFromCellID(location))
	}

	sort.Slice(ordered, func(i, j int) bool {
		iId := s2.CellIDFromToken(vicinity[ordered[i]].(string))
		jId := s2.CellIDFromToken(vicinity[ordered[j]].(string))

		iCell := s2.CellFromCellID(iId)
		jCell := s2.CellFromCellID(jId)

		iDistSum := 0.
		jDistSum := 0.
		for _, locationCell := range locationCells {
			iDistSum += utils.ChordAngleToKM(iCell.DistanceToCell(locationCell))
			jDistSum += utils.ChordAngleToKM(jCell.DistanceToCell(locationCell))
		}

		return iDistSum < jDistSum
	})

	if len(ordered) < 1 {
		return ""
	}

	return ordered[0]
}

func (a *system) getVicinity() map[string]s2.CellID {
	value, ok := a.env.GetMetric(metrics.MetricLocationInVicinity)
	if !ok {
		return nil
	}

	vicinityTokens := value.(map[string]interface{})
	vicinity := map[string]s2.CellID{}
	for nodeId, cellValue := range vicinityTokens {
		vicinity[nodeId] = s2.CellIDFromToken(cellValue.(string))
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

func (a *system) handleDeployments(deployment *deployment.Deployment) {
	time.Sleep(initialDeploymentDelay)

	timer := time.NewTimer(defaultInterval)

	for {
		<-timer.C

		log.Debugf("evaluating deployment %s", deployment.DeploymentId)

		action := deployment.GenerateAction()
		if action != nil {
			log.Debugf("generated action of type %s for deployment %s", action.GetActionId(), deployment.DeploymentId)
			a.performAction(action)
		}
		timer.Reset(defaultInterval)
	}
}

func (a *system) performAction(action actions.Action) {
	switch assertedAction := action.(type) {
	case *actions.RedirectAction:
		assertedAction.Execute(a.archimedesClient)
	case *actions.ExtendDeploymentAction:
		if assertedAction.IsExploring() {
			id := assertedAction.GetDeploymentId() + "_" + assertedAction.GetTarget()
			exploreChan := make(exploringMapValue)
			log.Debugf("exploring child %s", id)
			a.exploring.Store(id, exploreChan)
			go a.waitToBlacklist(assertedAction.GetDeploymentId(), assertedAction.GetTarget(), exploreChan)
		}
		assertedAction.Execute(a.deployerClient)
	case *actions.MigrateAction:
		assertedAction.Execute(a.deployerClient)
	case *actions.MultipleExtendDeploymentAction:
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
	id := deploymentId + "_" + childId
	value, ok := a.exploring.Load(id)
	if !ok {
		log.Debugf("missing %s", id)
		return false
	}

	a.exploring.Delete(id)

	exploreChan := value.(exploringMapValue)
	close(exploreChan)

	return true
}

func (a *system) waitToBlacklist(deploymentId, childId string, exploredChan <-chan interface{}) {
	value, ok := a.deployments.Load(deploymentId)
	if !ok {
		return
	}

	auxDeployment := value.(deploymentsMapValue)

	timer := time.NewTimer(blacklistInterval)
	select {
	case <-exploredChan:
		log.Debugf("exploring %s through %s was a success", deploymentId, childId)
	case <-timer.C:
		auxDeployment.BlacklistNode(childId)
	}
}
