package deployer

import (
	"encoding/json"
	"net/http"
	"strconv"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	log "github.com/sirupsen/logrus"
)

func deadChildHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)
	deadChildId := utils.ExtractPathVar(r, nodeIdPathVar)

	_, ok := suspectedChild.Load(deadChildId)
	if ok {
		log.Debugf("%s deployment from %s reported as dead, but ignored, already negotiating", deploymentId,
			deadChildId)
		return
	}

	grandchild := &utils.Node{}
	err := json.NewDecoder(r.Body).Decode(grandchild)
	if err != nil {
		panic(err)
	}

	log.Debugf("grandchild %s reported deployment %s from %s as dead", grandchild.Id, deploymentId, deadChildId)
	suspectedChild.Store(deadChildId, nil)
	hTable.removeChild(deploymentId, deadChildId)
	children.Delete(deadChildId)

	go attemptToExtend(deploymentId, "", grandchild, 0)
}

func takeChildHandler(w http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)

	child := &utils.Node{}
	err := json.NewDecoder(r.Body).Decode(child)
	if err != nil {
		panic(err)
	}

	log.Debugf("told to accept %s as child for deployment %s", child.Id, deploymentId)

	req := utils.BuildRequest(http.MethodPost, child.Addr+":"+strconv.Itoa(deployer.Port),
		api.GetImYourParentPath(deploymentId), myself)
	status, _ := utils.DoRequest(httpClient, req, nil)
	if status != http.StatusOK {
		log.Errorf("got status %d while telling %s that im his parent", status, child.Id)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	hTable.addChild(deploymentId, child)
	children.Store(child.Id, child)
}

func iAmYourParentHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)

	parent := &utils.Node{}
	err := json.NewDecoder(r.Body).Decode(parent)
	if err != nil {
		panic(err)
	}

	log.Debugf("told to accept %s as parent for deployment %s", parent.Id, deploymentId)

	hTable.setDeploymentParent(deploymentId, parent)
}

func getHierarchyTableHandler(w http.ResponseWriter, _ *http.Request) {
	utils.SendJSONReplyOK(w, hTable.toDTO())
}

func parentAliveHandler(_ http.ResponseWriter, r *http.Request) {
	parentId := utils.ExtractPathVar(r, nodeIdPathVar)
	log.Debugf("parent %s is alive", parentId)
	pTable.setParentUp(parentId)
}
