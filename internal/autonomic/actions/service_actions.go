package actions

import (
	"net/http"
	"strconv"

	internalUtils "github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
)

const (
	RedirectClientsId = "ACTION_REDIRECT_CLIENTS"
)

type RedirectAction struct {
	*actionWithDeploymentOriginTarget
	ErrorRedirectingCallback func()
}

func NewRedirectAction(deploymentId string, from, to *internalUtils.Node, amount int) *RedirectAction {
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

func (r *RedirectAction) Execute(client utils.GenericClient) {
	assertedClient := client.(archimedes.Client)
	assertedClient.SetHostPort(r.GetTarget().Addr + ":" + strconv.Itoa(internalUtils.ArchimedesPort))
	status := assertedClient.WillRedirectToYou(r.GetDeploymentId(), r.GetOrigin().Id)
	if status != http.StatusOK {
		return
	}

	assertedClient.SetHostPort(r.GetOrigin().Addr + ":" + strconv.Itoa(internalUtils.ArchimedesPort))
	status = assertedClient.Redirect(r.GetDeploymentId(), r.GetTarget().Addr, r.GetAmount())
	if status != http.StatusOK {
		r.getErrorRedirectingCallback()()
	}
}
