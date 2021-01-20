package actions

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/servers"
	"github.com/golang/geo/s2"
	log "github.com/sirupsen/logrus"
)

const (
	ExtendDeploymentID         = "ACTION_EXTEND_DEPLOYMENT"
	RemoveDeploymentID         = "ACTION_REMOVE_DEPLOYMENT"
	MultipleExtendDeploymentID = "ACTION_MULTIPLE_EXTEND_DEPLOYMENT"
)

type RemoveDeploymentAction struct {
	*actionWithDeployment
}

func NewRemoveDeploymentAction(deploymentID string) *RemoveDeploymentAction {
	return &RemoveDeploymentAction{
		actionWithDeployment: newActionWithDeployment(RemoveDeploymentID, deploymentID),
	}
}

func (m *RemoveDeploymentAction) Execute(addr string, client utils.GenericClient) {
	deployerClient := client.(deployer.Client)

	status := deployerClient.DeleteDeployment(addr, m.getDeploymentID())
	if status != http.StatusOK {
		log.Errorf("got status %d while attempting to delete deployment %s", status, m.getDeploymentID())
	}
}

type ExtendDeploymentAction struct {
	*actionWithDeploymentTarget
	deplFactory deployer.ClientFactory
}

func NewExtendDeploymentAction(deploymentID string, target *utils.Node, exploringTTL int,
	children []*utils.Node, location s2.CellID, toExclude map[string]interface{},
	setNodeExploringCallback func(nodeId string), deplFactory deployer.ClientFactory) *ExtendDeploymentAction {
	return &ExtendDeploymentAction{
		actionWithDeploymentTarget: newActionWithDeploymentTarget(ExtendDeploymentID, deploymentID, target,
			exploringTTL, children, location, toExclude, setNodeExploringCallback),
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

func (m *ExtendDeploymentAction) Execute(addr string, client utils.GenericClient) {
	log.Debugf("executing %s to %s", m.ActionID, m.GetTarget())

	deployerClient := client.(deployer.Client)
	addr = m.GetTarget().Addr + ":" + strconv.Itoa(deployer.Port)
	targetClient := m.deplFactory.New()

	has, _ := targetClient.HasDeployment(addr, m.getDeploymentID())
	if has {
		log.Debugf("%s already has deployment %s", m.GetTarget(), m.getDeploymentID())

		return
	}

	config := &api.ExtendDeploymentConfig{
		Children:  m.getChildren(),
		Locations: []s2.CellID{m.getLocation()},
		ToExclude: m.getToExclude(),
	}

	status := deployerClient.ExtendDeploymentTo(addr, m.getDeploymentID(), m.GetTarget(), m.exploringTTL(), config)
	if status != http.StatusOK {
		log.Errorf("got status code %d while extending deployment", status)

		return
	}

	if m.exploringTTL() != api.NotExploringTTL {
		m.getSetNodeAsExploringCallback()(m.GetTarget().ID)
	}
}

type MultipleExtendDeploymentAction struct {
	*actionWithDeploymentTargets
	deplFactory deployer.ClientFactory
}

func NewMultipleExtendDeploymentAction(deploymentID string, targets []*utils.Node, locations map[string][]s2.CellID,
	targetsExploring map[string]int, centroidsExtendedCallback func(centroid s2.CellID),
	toExclude map[string]interface{}, setNodeExploringCallback func(nodeId string),
	deplFactory deployer.ClientFactory) *MultipleExtendDeploymentAction {
	return &MultipleExtendDeploymentAction{
		actionWithDeploymentTargets: newActionWithDeploymentTargets(MultipleExtendDeploymentID, deploymentID,
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

func (m *MultipleExtendDeploymentAction) Execute(_ string, client utils.GenericClient) {
	targets := m.getTargets()

	var sb strings.Builder
	for _, target := range targets {
		sb.WriteString(fmt.Sprintf("%+v", target))
	}

	log.Debugf("executing %s to %v", m.ActionID, sb.String())

	deployerClient := client.(deployer.Client)
	locations := m.getLocations()
	extendedCentroidCallback := m.getCentroidCallback()
	targetsExploring := m.getTargetsExploring()
	toExclude := m.getToExclude()

	for _, target := range m.getTargets() {
		addr := target.Addr + ":" + strconv.Itoa(deployer.Port)

		has, _ := deployerClient.HasDeployment(addr, m.getDeploymentID())
		if has {
			log.Debugf("%s already has deployment %s", target, m.getDeploymentID())

			continue
		}

		config := &api.ExtendDeploymentConfig{
			Children:  nil,
			Locations: locations[target.ID],
			ToExclude: toExclude,
		}

		status := deployerClient.ExtendDeploymentTo(servers.DeployerLocalHostPort, m.getDeploymentID(), target,
			targetsExploring[target.ID], config)
		if status != http.StatusOK {
			log.Errorf("got status code %d while extending deployment", status)

			return
		}

		if targetsExploring[target.ID] != api.NotExploringTTL {
			m.getSetNodeAsExploringCallback()(target.ID)
		}

		for _, centroid := range locations[target.ID] {
			extendedCentroidCallback(centroid)
		}
	}
}
