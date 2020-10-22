package actions

import (
	"net/http"
	"strconv"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	publicUtils "github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	log "github.com/sirupsen/logrus"
)

const (
	ExtendServiceId  = "ACTION_EXTEND_SERVICE"
	RemoveServiceId  = "ACTION_REMOVE_SERVICE"
	MigrateServiceId = "ACTION_MIGRATE_SERVICE"
)

type RemoveServiceAction struct {
	*actionWithServiceTarget
}

func NewRemoveServiceAction(serviceId, target string) *RemoveServiceAction {
	return &RemoveServiceAction{
		actionWithServiceTarget: newActionWithServiceTarget(RemoveServiceId, serviceId, target),
	}
}

func (m *RemoveServiceAction) Execute(client utils.Client) {
	deployerClient := client.(deployer.Client)
	deployerClient.ShortenDeploymentFrom(m.GetServiceId(), m.GetTarget())
}

type ExtendServiceAction struct {
	*actionWithServiceTarget
}

func NewExtendServiceAction(serviceId, target string, exploring bool, parent *utils.Node,
	children []*utils.Node, location *publicUtils.Location) *ExtendServiceAction {
	return &ExtendServiceAction{
		actionWithServiceTarget: newActionWithServiceTarget(ExtendServiceId, serviceId, target, exploring, parent,
			children, location),
	}
}

func (m *ExtendServiceAction) IsExploring() bool {
	return m.Args[2].(bool)
}

func (m *ExtendServiceAction) GetParent() *utils.Node {
	return m.Args[3].(*utils.Node)
}

func (m *ExtendServiceAction) GetChildren() []*utils.Node {
	return m.Args[4].([]*utils.Node)
}

func (m *ExtendServiceAction) GetLocation() *publicUtils.Location {
	return m.Args[5].(*publicUtils.Location)
}

func (m *ExtendServiceAction) Execute(client utils.Client) {
	log.Debugf("executing %s to %s", m.ActionId, m.GetTarget())
	deployerClient := client.(*deployer.Client)

	targetClient := deployer.NewDeployerClient(m.GetTarget() + ":" + strconv.Itoa(deployer.Port))
	has, _ := targetClient.HasService(m.GetServiceId())
	if has {
		log.Debugf("%s already has service %s", m.GetTarget(), m.GetServiceId())
		return
	}

	if m.IsExploring() {
		archClient := archimedes.NewArchimedesClient(m.GetTarget() + ":" + strconv.Itoa(archimedes.Port))
		archClient.SetExploringClientLocation(m.GetServiceId(), m.GetLocation())
	}

	status := deployerClient.ExtendDeploymentTo(m.GetServiceId(), m.GetTarget(), m.GetParent(), m.GetChildren(),
		m.IsExploring())
	if status != http.StatusOK {
		log.Errorf("got status code %d while extending deployment", status)
		return
	}
}

type MigrateAction struct {
	*actionWithServiceOriginTarget
}

func NewMigrateAction(serviceId, from, to string) *MigrateAction {
	return &MigrateAction{
		actionWithServiceOriginTarget: newActionWithServiceOriginTarget(MigrateServiceId, serviceId, from, to),
	}
}

func (m *MigrateAction) Execute(client utils.Client) {
	log.Debugf("executing %s from %s to %s", MigrateServiceId, m.GetOrigin(), m.GetTarget())
	deployerClient := client.(*deployer.Client)
	status := deployerClient.MigrateDeployment(m.GetServiceId(), m.GetOrigin(), m.GetTarget())
	if status == http.StatusOK {
		log.Errorf("got status code %d while extending deployment", status)
	}
}
