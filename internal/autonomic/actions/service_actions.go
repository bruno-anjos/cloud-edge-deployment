package actions

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
)

const (
	RedirectClientsId = "ACTION_REDIRECT_CLIENTS"

	raAmountIndex = 3
)

type RedirectAction struct {
	*actionWithServiceOriginTarget
}

func NewRedirectAction(serviceId, from, to string, amount int) *RedirectAction {
	return &RedirectAction{
		actionWithServiceOriginTarget: newActionWithServiceOriginTarget(RedirectClientsId, serviceId, from, to,
			amount),
	}
}

func (r *RedirectAction) GetAmount() int {
	return r.Args[raAmountIndex].(int)
}

func (r *RedirectAction) Execute(client utils.Client) {
	assertedClient := client.(*archimedes.Client)
	assertedClient.SetHostPort(r.GetOrigin())
	assertedClient.Redirect(r.GetServiceId(), r.GetTarget(), r.GetAmount())
}
