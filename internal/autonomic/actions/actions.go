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

func newBasicAction(actionId string) *basicAction {
	return &basicAction{
		ActionId: actionId,
	}
}

func (b *basicAction) GetActionId() string {
	return b.ActionId
}

type actionWithArgs struct {
	*basicAction
	Args []interface{}
}

func newActionWithArgs(actionId string, args ...interface{}) *actionWithArgs {
	return &actionWithArgs{
		basicAction: newBasicAction(actionId),
		Args:        args,
	}
}

type actionWithServiceTarget struct {
	*actionWithArgs
}

func newActionWithServiceTarget(actionId, serviceId, target string,
	args ...interface{}) *actionWithServiceTarget {
	return &actionWithServiceTarget{
		actionWithArgs: newActionWithArgs(actionId, serviceId, target, args),
	}
}

func (a *actionWithServiceTarget) GetServiceId() string {
	return a.Args[0].(string)
}

func (a *actionWithServiceTarget) GetTarget() string {
	return a.Args[1].(string)
}

type actionWithServiceOriginTarget struct {
	*actionWithServiceTarget
}

func newActionWithServiceOriginTarget(actionId, serviceId, origin, target string,
	args ...interface{}) *actionWithServiceOriginTarget {
	return &actionWithServiceOriginTarget{
		actionWithServiceTarget: newActionWithServiceTarget(actionId, serviceId, target, origin, args),
	}
}

func (a *actionWithServiceOriginTarget) GetOrigin() string {
	return a.Args[2].(string)
}
