package actions

import (
	"net/http"
	"strconv"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
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
	children []*utils.Node) *ExtendServiceAction {
	return &ExtendServiceAction{
		actionWithServiceTarget: newActionWithServiceTarget(ExtendServiceId, serviceId, target, exploring, parent,
			children),
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

func (m *ExtendServiceAction) Execute(client utils.Client) {
	log.Debugf("executing %s to %s", m.ActionId, m.GetTarget())
	deployerClient := client.(*deployer.Client)

	targetClient := deployer.NewDeployerClient(m.GetTarget() + ":" + strconv.Itoa(deployer.Port))
	has, _ := targetClient.HasService(m.GetServiceId())
	if has {
		log.Debugf("%s already has service %s", m.GetTarget(), m.GetServiceId())
		return
	}

	status := deployerClient.ExtendDeploymentTo(m.GetServiceId(), m.GetTarget(), m.GetParent(), m.GetChildren())
	if status != http.StatusOK {
		log.Errorf("got status code %d while extending deployment", status)
		return
	}

	if m.IsExploring() {
		status = deployerClient.SetExploring(m.GetServiceId(), m.GetTarget())
		if status != http.StatusOK {
			log.Errorf("got status %d while setting %s as exploring deployment %s", status, m.GetTarget(),
				m.GetServiceId())
		}
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
