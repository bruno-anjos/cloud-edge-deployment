package actions

import (
	"net/http"
	"strconv"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
)

const (
	RedirectClientsId = "ACTION_REDIRECT_CLIENTS"
)

type RedirectAction struct {
	*actionWithDeploymentOriginTarget
	ErrorRedirectingCallback func()
}

func NewRedirectAction(deploymentId, from, to string, amount int) *RedirectAction {
	return &RedirectAction{
		actionWithDeploymentOriginTarget: newActionWithDeploymentOriginTarget(RedirectClientsId, deploymentId, from, to,
			amount),
	}
}

func (r *RedirectAction) GetAmount() int {
	return r.Args[3].(int)
}

func (r *RedirectAction) SetErrorRedirectingCallback(errorRedirectingCallback func()) {
	r.ErrorRedirectingCallback = errorRedirectingCallback
}

func (r *RedirectAction) getErrorRedirectingCallback() func() {
	return r.ErrorRedirectingCallback
}

func (r *RedirectAction) Execute(client utils.Client) {
	assertedClient := client.(*archimedes.Client)
	assertedClient.SetHostPort(r.GetTarget() + ":" + strconv.Itoa(archimedes.Port))
	status := assertedClient.WillRedirectToYou(r.GetDeploymentId(), r.GetOrigin())
	if status != http.StatusOK {
		return
	}

	assertedClient.SetHostPort(r.GetOrigin() + ":" + strconv.Itoa(archimedes.Port))
	status = assertedClient.Redirect(r.GetDeploymentId(), r.GetTarget(), r.GetAmount())
	if status != http.StatusOK {
		r.getErrorRedirectingCallback()()
	}
}
