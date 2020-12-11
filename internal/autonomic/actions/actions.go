package actions

import (
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
)

type Action interface {
	GetActionID() string
	Execute(client utils.GenericClient)
}

type basicAction struct {
	ActionID string
}

func newBasicAction(actionID string) *basicAction {
	return &basicAction{
		ActionID: actionID,
	}
}

func (b *basicAction) GetActionID() string {
	return b.ActionID
}

type actionWithArgs struct {
	*basicAction
	Args []interface{}
}

func newActionWithArgs(actionID string, args ...interface{}) *actionWithArgs {
	return &actionWithArgs{
		basicAction: newBasicAction(actionID),
		Args:        args,
	}
}

type actionWithDeployment struct {
	*actionWithArgs
}

func newActionWithDeployment(actionID, deploymentID string, args ...interface{}) *actionWithDeployment {
	newArgs := []interface{}{deploymentID}
	newArgs = append(newArgs, args...)

	return &actionWithDeployment{
		actionWithArgs: newActionWithArgs(actionID, newArgs...),
	}
}

func (a *actionWithDeployment) getDeploymentID() string {
	return a.Args[0].(string)
}

type actionWithDeploymentTarget struct {
	*actionWithDeployment
}

func newActionWithDeploymentTarget(actionID, deploymentID string, target *utils.Node,
	args ...interface{}) *actionWithDeploymentTarget {
	newArgs := []interface{}{target}
	newArgs = append(newArgs, args...)

	return &actionWithDeploymentTarget{
		actionWithDeployment: newActionWithDeployment(actionID, deploymentID, newArgs...),
	}
}

func (a *actionWithDeploymentTarget) GetTarget() *utils.Node {
	return a.Args[1].(*utils.Node)
}

type actionWithDeploymentTargets struct {
	*actionWithDeployment
}

func newActionWithDeploymentTargets(actionID, deploymentID string, targets []*utils.Node,
	args ...interface{}) *actionWithDeploymentTargets {
	newArgs := []interface{}{targets}
	newArgs = append(newArgs, args...)

	return &actionWithDeploymentTargets{
		actionWithDeployment: newActionWithDeployment(actionID, deploymentID, newArgs...),
	}
}

func (a *actionWithDeploymentTargets) getTargets() []*utils.Node {
	return a.Args[1].([]*utils.Node)
}

type actionWithDeploymentOriginTarget struct {
	*actionWithDeploymentTarget
}

func newActionWithDeploymentOriginTarget(actionID, deploymentID string, origin, target *utils.Node,
	args ...interface{}) *actionWithDeploymentOriginTarget {
	newArgs := make([]interface{}, len(args)+1)

	newArgs[0] = origin
	for i, arg := range args {
		newArgs[i+1] = arg
	}

	return &actionWithDeploymentOriginTarget{
		actionWithDeploymentTarget: newActionWithDeploymentTarget(actionID, deploymentID, target, newArgs...),
	}
}

func (a *actionWithDeploymentOriginTarget) GetOrigin() *utils.Node {
	return a.Args[2].(*utils.Node)
}
