package actions

import (
	"net/http"
	"strconv"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	log "github.com/sirupsen/logrus"
)

const (
	AddServiceId     = "ACTION_ADD_SERVICE"
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

type AddServiceAction struct {
	*actionWithServiceTarget
	Exploring bool
}

func NewAddServiceAction(serviceId, target string, exploring bool) *AddServiceAction {
	return &AddServiceAction{
		actionWithServiceTarget: newActionWithServiceTarget(AddServiceId, serviceId, target),
		Exploring:               exploring,
	}
}

func (m *AddServiceAction) Execute(client utils.Client) {
	log.Debugf("executing %s to %s", m.ActionId, m.GetTarget())
	deployerClient := client.(*deployer.Client)

	targetClient := deployer.NewDeployerClient(m.GetTarget() + ":" + strconv.Itoa(deployer.Port))
	has, _ := targetClient.HasService(m.GetServiceId())
	if has {
		log.Debugf("%s already has service %s", m.GetTarget(), m.GetServiceId())
		return
	}

	status := deployerClient.ExtendDeploymentTo(m.GetServiceId(), m.GetTarget())
	if status != http.StatusOK {
		log.Errorf("got status code %d while extending deployment", status)
		return
	}

	if m.Exploring {
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
