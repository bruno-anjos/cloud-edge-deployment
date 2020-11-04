package actions

import (
	"net/http"
	"strconv"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/golang/geo/s2"
	log "github.com/sirupsen/logrus"
)

const (
	ExtendDeploymentId         = "ACTION_EXTEND_DEPLOYMENT"
	RemoveDeploymentId         = "ACTION_REMOVE_DEPLOYMENT"
	MigrateDeploymentId        = "ACTION_MIGRATE_DEPLOYMENT"
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

func (m *RemoveDeploymentAction) Execute(client utils.Client) {
	deployerClient := client.(*deployer.Client)
	status := deployerClient.DeleteDeployment(m.GetDeploymentId())
	if status != http.StatusOK{
		log.Errorf("got status %d while attempting to delete deployment %s", status, m.GetDeploymentId())
	}
}

type ExtendDeploymentAction struct {
	*actionWithDeploymentTarget
}

func NewExtendDeploymentAction(deploymentId, target string, exploring bool, parent *utils.Node,
	children []*utils.Node, location s2.CellID) *ExtendDeploymentAction {
	return &ExtendDeploymentAction{
		actionWithDeploymentTarget: newActionWithDeploymentTarget(ExtendDeploymentId, deploymentId, target, exploring, parent,
			children, location),
	}
}

func (m *ExtendDeploymentAction) IsExploring() bool {
	return m.Args[2].(bool)
}

func (m *ExtendDeploymentAction) GetParent() *utils.Node {
	return m.Args[3].(*utils.Node)
}

func (m *ExtendDeploymentAction) GetChildren() []*utils.Node {
	return m.Args[4].([]*utils.Node)
}

func (m *ExtendDeploymentAction) GetLocation() s2.CellID {
	return m.Args[5].(s2.CellID)
}

func (m *ExtendDeploymentAction) Execute(client utils.Client) {
	log.Debugf("executing %s to %s", m.ActionId, m.GetTarget())
	deployerClient := client.(*deployer.Client)

	targetClient := deployer.NewDeployerClient(m.GetTarget() + ":" + strconv.Itoa(deployer.Port))
	has, _ := targetClient.HasDeployment(m.GetDeploymentId())
	if has {
		log.Debugf("%s already has deployment %s", m.GetTarget(), m.GetDeploymentId())
		return
	}

	status := deployerClient.ExtendDeploymentTo(m.GetDeploymentId(), m.GetTarget(), m.GetParent(),
		[]s2.CellID{m.GetLocation()}, m.GetChildren(), m.IsExploring())
	if status != http.StatusOK {
		log.Errorf("got status code %d while extending deployment", status)
		return
	}
}

type MultipleExtendDeploymentAction struct {
	*actionWithDeploymentTargets
}

func NewMultipleExtendDeploymentAction(deploymentId string, targets []string, parent *utils.Node,
	locations map[string][]s2.CellID, targetsExploring map[string]bool,
	centroidsExtendedCallback func(centroid s2.CellID)) *MultipleExtendDeploymentAction {
	return &MultipleExtendDeploymentAction{
		actionWithDeploymentTargets: newActionWithDeploymentTargets(MultipleExtendDeploymentId, deploymentId,
			targets, parent, locations, targetsExploring, centroidsExtendedCallback),
	}
}

func (m *MultipleExtendDeploymentAction) GetParent() *utils.Node {
	return m.Args[2].(*utils.Node)
}

func (m *MultipleExtendDeploymentAction) GetLocations() map[string][]s2.CellID {
	return m.Args[3].(map[string][]s2.CellID)
}

func (m *MultipleExtendDeploymentAction) GetTargetsExploring() map[string]bool {
	return m.Args[4].(map[string]bool)
}

func (m *MultipleExtendDeploymentAction) GetCentroidCallback() func(centroid s2.CellID) {
	return m.Args[5].(func(centroid s2.CellID))
}

func (m *MultipleExtendDeploymentAction) Execute(client utils.Client) {
	log.Debugf("executing %s to %+v", m.ActionId, m.GetTargets())
	deployerClient := client.(*deployer.Client)
	locations := m.GetLocations()
	extendedCentroidCallback := m.GetCentroidCallback()
	targetsExploring := m.GetTargetsExploring()

	for _, target := range m.GetTargets() {
		targetClient := deployer.NewDeployerClient(target + ":" + strconv.Itoa(deployer.Port))
		has, _ := targetClient.HasDeployment(m.GetDeploymentId())
		if has {
			log.Debugf("%s already has deployment %s", target, m.GetDeploymentId())
			continue
		}

		status := deployerClient.ExtendDeploymentTo(m.GetDeploymentId(), target, m.GetParent(), locations[target],
			nil, targetsExploring[target])
		if status != http.StatusOK {
			log.Errorf("got status code %d while extending deployment", status)
			return
		}

		for _, centroid := range locations[target] {
			extendedCentroidCallback(centroid)
		}
	}

}

type MigrateAction struct {
	*actionWithDeploymentOriginTarget
}

func NewMigrateAction(deploymentId, from, to string) *MigrateAction {
	return &MigrateAction{
		actionWithDeploymentOriginTarget: newActionWithDeploymentOriginTarget(MigrateDeploymentId, deploymentId, from, to),
	}
}

func (m *MigrateAction) Execute(client utils.Client) {
	log.Debugf("executing %s from %s to %s", MigrateDeploymentId, m.GetOrigin(), m.GetTarget())
	deployerClient := client.(*deployer.Client)
	status := deployerClient.MigrateDeployment(m.GetDeploymentId(), m.GetOrigin(), m.GetTarget())
	if status == http.StatusOK {
		log.Errorf("got status code %d while extending deployment", status)
	}
}
