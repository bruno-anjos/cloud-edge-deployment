package actions

func NewRedirectAction(to string) *ActionWithArgs {
	return NewActionWithArgs(REDIRECT_CLIENTS_ID, to)
}

const (
	REDIRECT_CLIENTS_ID = "ACTION_REDIRECT_CLIENTS"
)
