package actions

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

type Action interface {
	GetActionId() string
	Execute(client utils.Client)
}

type basicAction struct {
	ActionId string
}

func NewBasicAction(actionId string) *basicAction {
	return &basicAction{
		ActionId: actionId,
	}
}

func (b *basicAction) GetActionId() string {
	return b.ActionId
}

type ActionWithArgs struct {
	*basicAction
	Args []interface{}
}

func NewActionWithArgs(actionId string, args ...interface{}) *ActionWithArgs {
	return &ActionWithArgs{
		basicAction: NewBasicAction(actionId),
		Args:        args,
	}
}

type ActionWithServiceTarget struct {
	*ActionWithArgs
}

func NewActionWithServiceTarget(actionId, serviceId, target string,
	args ...interface{}) *ActionWithServiceTarget {
	return &ActionWithServiceTarget{
		ActionWithArgs: NewActionWithArgs(actionId, serviceId, target, args),
	}
}

func (a *ActionWithServiceOriginTarget) GetServiceId() string {
	return a.Args[0].(string)
}

func (a *ActionWithServiceOriginTarget) GetTarget() string {
	return a.Args[1].(string)
}

type ActionWithServiceOriginTarget struct {
	*ActionWithServiceTarget
}

func NewActionWithServiceOriginTarget(actionId, serviceId, origin, target string,
	args ...interface{}) *ActionWithServiceOriginTarget {
	return &ActionWithServiceOriginTarget{
		ActionWithServiceTarget: NewActionWithServiceTarget(actionId, serviceId, target, origin, args),
	}
}

func (a *ActionWithServiceOriginTarget) GetOrigin() string {
	return a.Args[2].(string)
}
