package actions

import (
	"net/http"
	"strconv"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"

	"github.com/golang/geo/s2"
	log "github.com/sirupsen/logrus"
)

const (
	ExtendDeploymentId         = "ACTION_EXTEND_DEPLOYMENT"
	RemoveDeploymentId         = "ACTION_REMOVE_DEPLOYMENT"
	MultipleExtendDeploymentId = "ACTION_MULTIPLE_EXTEND_DEPLOYMENT"
)

type RemoveDeploymentAction struct {
	*actionWithDeployment
}

func NewRemoveDeploymentAction(deploymentId string) *RemoveDeploymentAction {
	return &RemoveDeploymentAction{
		actionWithDeployment: newActionWithDeployment(RemoveDeploymentId, deploymentId),
	}
}

func (m *RemoveDeploymentAction) Execute(client utils.GenericClient) {
	deployerClient := client.(deployer.Client)
	status := deployerClient.DeleteDeployment(m.GetDeploymentId())
	if status != http.StatusOK {
		log.Errorf("got status %d while attempting to delete deployment %s", status, m.GetDeploymentId())
	}
}

type ExtendDeploymentAction struct {
	*actionWithDeploymentTarget
	deplFactory deployer.ClientFactory
}

func NewExtendDeploymentAction(deploymentId string, target *utils.Node, exploringTTL int,
	children []*utils.Node, location s2.CellID, toExclude map[string]interface{},
	setNodeExploringCallback func(nodeId string), deplFactory deployer.ClientFactory) *ExtendDeploymentAction {
	return &ExtendDeploymentAction{
		actionWithDeploymentTarget: newActionWithDeploymentTarget(ExtendDeploymentId, deploymentId, target, exploringTTL,
			children, location, toExclude, setNodeExploringCallback),
		deplFactory: deplFactory,
	}
}

func (m *ExtendDeploymentAction) exploringTTL() int {
	return m.Args[2].(int)
}

func (m *ExtendDeploymentAction) getChildren() []*utils.Node {
	return m.Args[3].([]*utils.Node)
}

func (m *ExtendDeploymentAction) getLocation() s2.CellID {
	return m.Args[4].(s2.CellID)
}

func (m *ExtendDeploymentAction) getToExclude() map[string]interface{} {
	return m.Args[5].(map[string]interface{})
}

func (m *ExtendDeploymentAction) getSetNodeAsExploringCallback() func(nodeId string) {
	return m.Args[6].(func(nodeId string))
}

func (m *ExtendDeploymentAction) Execute(client utils.GenericClient) {
	log.Debugf("executing %s to %s", m.ActionId, m.GetTarget())
	deployerClient := client.(deployer.Client)

	targetClient := m.deplFactory.New(m.GetTarget().Addr + ":" + strconv.Itoa(deployer.Port))
	has, _ := targetClient.HasDeployment(m.GetDeploymentId())
	if has {
		log.Debugf("%s already has deployment %s", m.GetTarget(), m.GetDeploymentId())
		return
	}

	config := &api.ExtendDeploymentConfig{
		Children:  m.getChildren(),
		Locations: []s2.CellID{m.getLocation()},
		ToExclude: m.getToExclude(),
	}
	status := deployerClient.ExtendDeploymentTo(m.GetDeploymentId(), m.GetTarget(), m.exploringTTL(), config)
	if status != http.StatusOK {
		log.Errorf("got status code %d while extending deployment", status)
		return
	}

	if m.exploringTTL() != api.NotExploringTTL {
		m.getSetNodeAsExploringCallback()(m.GetTarget().Id)
	}
}

type MultipleExtendDeploymentAction struct {
	*actionWithDeploymentTargets
	deplFactory deployer.ClientFactory
}

func NewMultipleExtendDeploymentAction(deploymentId string, targets []*utils.Node, locations map[string][]s2.CellID,
	targetsExploring map[string]int, centroidsExtendedCallback func(centroid s2.CellID),
	toExclude map[string]interface{}, setNodeExploringCallback func(nodeId string),
	deplFactory deployer.ClientFactory) *MultipleExtendDeploymentAction {
	return &MultipleExtendDeploymentAction{
		actionWithDeploymentTargets: newActionWithDeploymentTargets(MultipleExtendDeploymentId, deploymentId,
			targets, locations, targetsExploring, centroidsExtendedCallback, toExclude, setNodeExploringCallback),
		deplFactory: deplFactory,
	}
}

func (m *MultipleExtendDeploymentAction) getLocations() map[string][]s2.CellID {
	return m.Args[2].(map[string][]s2.CellID)
}

func (m *MultipleExtendDeploymentAction) getTargetsExploring() map[string]int {
	return m.Args[3].(map[string]int)
}

func (m *MultipleExtendDeploymentAction) getCentroidCallback() func(centroid s2.CellID) {
	return m.Args[4].(func(centroid s2.CellID))
}

func (m *MultipleExtendDeploymentAction) getToExclude() map[string]interface{} {
	return m.Args[5].(map[string]interface{})
}

func (m *MultipleExtendDeploymentAction) getSetNodeAsExploringCallback() func(nodeId string) {
	return m.Args[6].(func(nodeId string))
}

func (m *MultipleExtendDeploymentAction) Execute(client utils.GenericClient) {
	log.Debugf("executing %s to %+v", m.ActionId, m.GetTargets())
	deployerClient := client.(deployer.Client)
	locations := m.getLocations()
	extendedCentroidCallback := m.getCentroidCallback()
	targetsExploring := m.getTargetsExploring()
	toExclude := m.getToExclude()

	for _, target := range m.GetTargets() {
		targetClient := m.deplFactory.New(target.Addr + ":" + strconv.Itoa(deployer.Port))
		has, _ := targetClient.HasDeployment(m.GetDeploymentId())
		if has {
			log.Debugf("%s already has deployment %s", target, m.GetDeploymentId())
			continue
		}

		config := &api.ExtendDeploymentConfig{
			Children:  nil,
			Locations: locations[target.Id],
			ToExclude: toExclude,
		}

		status := deployerClient.ExtendDeploymentTo(m.GetDeploymentId(), target, targetsExploring[target.Id], config)
		if status != http.StatusOK {
			log.Errorf("got status code %d while extending deployment", status)
			return
		}

		if targetsExploring[target.Id] != api.NotExploringTTL {
			m.getSetNodeAsExploringCallback()(target.Id)
		}

		for _, centroid := range locations[target.Id] {
			extendedCentroidCallback(centroid)
		}
	}

}
