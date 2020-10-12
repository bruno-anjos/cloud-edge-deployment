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
	ExploreId        = "ACTION_EXPLORE"
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
	ExploreChan chan struct{}
}

func NewAddServiceAction(serviceId, target string, exploreChan chan struct{}) *AddServiceAction {
	return &AddServiceAction{
		actionWithServiceTarget: newActionWithServiceTarget(AddServiceId, serviceId, target),
		ExploreChan:             exploreChan,
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

	if m.ExploreChan != nil {
		status = deployerClient.SetExploring(m.GetServiceId(), m.GetTarget())
		if status != http.StatusOK {
			log.Errorf("got status %d while setting %s as exploring deployment %s", status, m.GetTarget(),
				m.GetServiceId())
		}
	}

}

type ExploreAction struct {
	*actionWithServiceTarget
}

func NewExploreAction(serviceId, target string) *ExploreAction {
	return &ExploreAction{
		actionWithServiceTarget: newActionWithServiceTarget(ExploreId, serviceId, target),
	}
}

func (m *ExploreAction) Execute(client utils.Client) {
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
