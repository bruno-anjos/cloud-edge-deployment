package actions

import (
	internalUtils "github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
)

type Action interface {
	GetActionId() string
	Execute(client utils.GenericClient)
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

type actionWithDeployment struct {
	*actionWithArgs
}

func newActionWithDeployment(actionId, deploymentId string, args ...interface{}) *actionWithDeployment {
	newArgs := []interface{}{deploymentId}
	newArgs = append(newArgs, args...)

	return &actionWithDeployment{
		actionWithArgs: newActionWithArgs(actionId, newArgs...),
	}
}

func (a *actionWithDeployment) GetDeploymentId() string {
	return a.Args[0].(string)
}

type actionWithDeploymentTarget struct {
	*actionWithDeployment
}

func newActionWithDeploymentTarget(actionId, deploymentId string, target *internalUtils.Node,
	args ...interface{}) *actionWithDeploymentTarget {
	newArgs := []interface{}{target}
	newArgs = append(newArgs, args...)

	return &actionWithDeploymentTarget{
		actionWithDeployment: newActionWithDeployment(actionId, deploymentId, newArgs...),
	}
}

func (a *actionWithDeploymentTarget) GetTarget() *internalUtils.Node {
	return a.Args[1].(*internalUtils.Node)
}

type actionWithDeploymentTargets struct {
	*actionWithDeployment
}

func newActionWithDeploymentTargets(actionId, deploymentId string, targets []*internalUtils.Node,
	args ...interface{}) *actionWithDeploymentTargets {
	newArgs := []interface{}{targets}
	newArgs = append(newArgs, args...)

	return &actionWithDeploymentTargets{
		actionWithDeployment: newActionWithDeployment(actionId, deploymentId, newArgs...),
	}
}

func (a *actionWithDeploymentTargets) GetTargets() []*internalUtils.Node {
	return a.Args[1].([]*internalUtils.Node)
}

type actionWithDeploymentOriginTarget struct {
	*actionWithDeploymentTarget
}

func newActionWithDeploymentOriginTarget(actionId, deploymentId string, origin, target *internalUtils.Node,
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

func (a *actionWithDeploymentOriginTarget) GetOrigin() *internalUtils.Node {
	return a.Args[2].(*internalUtils.Node)
}
