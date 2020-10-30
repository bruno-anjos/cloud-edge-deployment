package actions

import (
	"net/http"
	"strconv"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/golang/geo/s2"
	log "github.com/sirupsen/logrus"
)

const (
	ExtendDeploymentId  = "ACTION_EXTEND_DEPLOYMENT"
	RemoveDeploymentId  = "ACTION_REMOVE_DEPLOYMENT"
	MigrateDeploymentId = "ACTION_MIGRATE_DEPLOYMENT"
)

type RemoveDeploymentAction struct {
	*actionWithDeploymentTarget
}

func NewRemoveDeploymentAction(deploymentId, target string) *RemoveDeploymentAction {
	return &RemoveDeploymentAction{
		actionWithDeploymentTarget: newActionWithDeploymentTarget(RemoveDeploymentId, deploymentId, target),
	}
}

func (m *RemoveDeploymentAction) Execute(client utils.Client) {
	deployerClient := client.(deployer.Client)
	deployerClient.ShortenDeploymentFrom(m.GetDeploymentId(), m.GetTarget())
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

	if m.IsExploring() {
		archClient := archimedes.NewArchimedesClient(m.GetTarget() + ":" + strconv.Itoa(archimedes.Port))
		archClient.SetExploringCells(m.GetDeploymentId(), m.GetLocation())
	}

	status := deployerClient.ExtendDeploymentTo(m.GetDeploymentId(), m.GetTarget(), m.GetParent(), m.GetLocation(),
		m.GetChildren(), m.IsExploring())
	if status != http.StatusOK {
		log.Errorf("got status code %d while extending deployment", status)
		return
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
