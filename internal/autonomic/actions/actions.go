package actions

type Action interface {
	GetActionId() string
}

type BasicAction struct {
	ActionId string
}

func NewBasicAction(actionId string) *BasicAction {
	return &BasicAction{
		ActionId: actionId,
	}
}

func (b *BasicAction) GetActionId() string {
	return b.ActionId
}

type ActionWithArgs struct {
	*BasicAction
	Args []string
}

func NewActionWithArgs(actionId string, args ...string) *ActionWithArgs {
	return &ActionWithArgs{
		BasicAction: &BasicAction{
			ActionId: actionId,
		},
		Args: args,
	}
}
