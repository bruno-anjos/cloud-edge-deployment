package actions

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
)

const (
	REDIRECT_CLIENTS_ID = "ACTION_REDIRECT_CLIENTS"

	raAmountIndex = 3
)

type RedirectAction struct {
	*ActionWithServiceOriginTarget
}

func NewRedirectAction(serviceId, from, to string, amountPercentage float64) *RedirectAction {
	return &RedirectAction{
		ActionWithServiceOriginTarget: NewActionWithServiceOriginTarget(REDIRECT_CLIENTS_ID, serviceId, from, to,
			amountPercentage),
	}
}

func (r *RedirectAction) GetAmount() int {
	return r.Args[raAmountIndex].(int)
}

func (r *RedirectAction) Execute(client utils.Client) {
	assertedClient := client.(*archimedes.Client)
	assertedClient.SetHostPort(r.GetOrigin(), archimedes.Port)
	assertedClient.Redirect(r.GetServiceId(), r.GetTarget(), r.GetAmount())
}
