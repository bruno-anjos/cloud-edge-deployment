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

type actionWithDeploymentTarget struct {
	*actionWithArgs
}

func newActionWithDeploymentTarget(actionId, deploymentId, target string,
	args ...interface{}) *actionWithDeploymentTarget {
	newArgs := []interface{}{deploymentId, target}
	newArgs = append(newArgs, args...)

	return &actionWithDeploymentTarget{
		actionWithArgs: newActionWithArgs(actionId, newArgs...),
	}
}

func (a *actionWithDeploymentTarget) GetDeploymentId() string {
	return a.Args[0].(string)
}

func (a *actionWithDeploymentTarget) GetTarget() string {
	return a.Args[1].(string)
}

type actionWithDeploymentOriginTarget struct {
	*actionWithDeploymentTarget
}

func newActionWithDeploymentOriginTarget(actionId, deploymentId, origin, target string,
	args ...interface{}) *actionWithDeploymentOriginTarget {
	newArgs := make([]interface{}, len(args)+1)
	newArgs[0] = origin
	for i, arg := range args {
		newArgs[i+1] = arg
	}

	return &actionWithDeploymentOriginTarget{
		actionWithDeploymentTarget: newActionWithDeploymentTarget(actionId, deploymentId, target, newArgs...),
	}
}

func (a *actionWithDeploymentOriginTarget) GetOrigin() string {
	return a.Args[2].(string)
}
