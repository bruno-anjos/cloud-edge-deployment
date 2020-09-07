package actions

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
)

const (
	ADD_SERVICE_ID     = "ACTION_ADD_SERVICE"
	REMOVE_SERVICE_ID  = "ACTION_REMOVE_SERVICE"
	MIGRATE_SERVICE_ID = "ACTION_MIGRATE_SERVICE"
)

type RemoveServiceAction struct {
	*actionWithServiceTarget
}

func NewRemoveServiceAction(serviceId, target string) *RemoveServiceAction {
	return &RemoveServiceAction{
		actionWithServiceTarget: newActionWithServiceTarget(REMOVE_SERVICE_ID, serviceId, target),
	}
}

func (m *RemoveServiceAction) Execute(client utils.Client) {
	deployerClient := client.(deployer.Client)
	deployerClient.ShortenDeploymentFrom(m.GetServiceId(), m.GetTarget())
}

type AddServiceAction struct {
	*actionWithServiceTarget
}

func NewAddServiceAction(serviceId, target string) *AddServiceAction {
	return &AddServiceAction{
		actionWithServiceTarget: newActionWithServiceTarget(ADD_SERVICE_ID, serviceId, target),
	}
}

func (m *AddServiceAction) Execute(client utils.Client) {
	deployerClient := client.(deployer.Client)
	deployerClient.ExtendDeploymentTo(m.GetServiceId(), m.GetTarget())
}

type MigrateAction struct {
	*actionWithServiceOriginTarget
}

func NewMigrateAction(serviceId, from, to string) *MigrateAction {
	return &MigrateAction{
		actionWithServiceOriginTarget: newActionWithServiceOriginTarget(MIGRATE_SERVICE_ID, serviceId, from, to),
	}
}

func (m *MigrateAction) Execute(client utils.Client) {
	deployerClient := client.(deployer.Client)
	deployerClient.MigrateDeployment(m.GetServiceId(), m.GetOrigin(), m.GetTarget())
}
