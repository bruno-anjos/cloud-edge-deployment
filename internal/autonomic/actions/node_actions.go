package actions

func NewAddServiceAction(to string) *ActionWithArgs {
	return NewActionWithArgs(ADD_SERVICE_ID, to)
}

func NewMigrateAction(to string) *ActionWithArgs {
	return NewActionWithArgs(MIGRATE_SERVICE_ID, to)
}

const (
	ADD_SERVICE_ID     = "ACTION_ADD_SERVICE"
	REMOVE_SERVICE_ID  = "ACTION_REMOVE_SERVICE"
	MIGRATE_SERVICE_ID = "ACTION_MIGRATE_SERVICE"
)

var (
	REMOVE_SERVICE_ACTION = NewBasicAction(REMOVE_SERVICE_ID)
)
