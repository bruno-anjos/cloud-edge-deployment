package actions

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

const (
	ADD_SERVICE_ID     = "ACTION_ADD_SERVICE"
	REMOVE_SERVICE_ID  = "ACTION_REMOVE_SERVICE"
	MIGRATE_SERVICE_ID = "ACTION_MIGRATE_SERVICE"
)

type RemoveServiceAction struct {
	*ActionWithServiceTarget
}

func NewRemoveServiceAction(serviceId, target string) *RemoveServiceAction {
	return &RemoveServiceAction{
		ActionWithServiceTarget: NewActionWithServiceTarget(REMOVE_SERVICE_ID, serviceId, target),
	}
}

func (m *RemoveServiceAction) Execute(client utils.Client) {
	// TODO call deployer to add service to target
}

type AddServiceAction struct {
	*ActionWithServiceTarget
}

func NewAddServiceAction(serviceId, target string) *AddServiceAction {
	return &AddServiceAction{
		ActionWithServiceTarget: NewActionWithServiceTarget(ADD_SERVICE_ID, serviceId, target),
	}
}

func (m *AddServiceAction) Execute(client utils.Client) {
	// TODO call deployer to add service to target
}

type MigrateAction struct {
	*ActionWithServiceOriginTarget
}

func NewMigrateAction(serviceId, from, to string) *MigrateAction {
	return &MigrateAction{
		ActionWithServiceOriginTarget: NewActionWithServiceOriginTarget(MIGRATE_SERVICE_ID, serviceId, from, to),
	}
}

func (m *MigrateAction) Execute(client utils.Client) {
	// TODO call deployer to migrate deployment
}
