package actions

import (
	"net/http"
	"strconv"

	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
)

const (
	RedirectClientsID = "ACTION_REDIRECT_CLIENTS"
)

type RedirectAction struct {
	*actionWithDeploymentOriginTarget
	ErrorRedirectingCallback func()
}

func NewRedirectAction(deploymentID string, from, to *utils.Node, amount int) *RedirectAction {
	return &RedirectAction{
		actionWithDeploymentOriginTarget: newActionWithDeploymentOriginTarget(RedirectClientsID, deploymentID, from, to,
			amount),
		ErrorRedirectingCallback: nil,
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

func (r *RedirectAction) Execute(client utils.GenericClient) {
	assertedClient := client.(archimedes.Client)
	assertedClient.SetHostPort(r.GetTarget().Addr + ":" + strconv.Itoa(archimedes.Port))

	status := assertedClient.WillRedirectToYou(r.getDeploymentID(), r.GetOrigin().ID)
	if status != http.StatusOK {
		return
	}

	assertedClient.SetHostPort(r.GetOrigin().Addr + ":" + strconv.Itoa(archimedes.Port))

	status = assertedClient.Redirect(r.getDeploymentID(), r.GetTarget().Addr, r.GetAmount())
	if status != http.StatusOK {
		r.getErrorRedirectingCallback()()
	}
}
