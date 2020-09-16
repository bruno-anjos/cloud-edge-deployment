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

	body := api.DeadChildRequestBody{}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		panic(err)
	}

	log.Debugf("grandchild %s reported deployment %s from %s as dead", body.Grandchild.Id, deploymentId, deadChildId)
	suspectedChild.Store(deadChildId, nil)
	hTable.removeChild(deploymentId, deadChildId)
	children.Delete(deadChildId)

	go attemptToExtend(deploymentId, "", body.Grandchild, 0, body.Alternatives)
}

func canTakeChildHandler(w http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)
	possibleChild := utils.ExtractPathVar(r, nodeIdPathVar)

	parent := hTable.getParent(deploymentId)
	if possibleChild != parent.Id {
		log.Debugf("can take child %s, parent is %s", possibleChild, parent.Id)
	} else {
		log.Debugf("rejecting child %s", possibleChild)
		w.WriteHeader(http.StatusConflict)
	}
}

func takeChildHandler(w http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)

	child := &utils.Node{}
	err := json.NewDecoder(r.Body).Decode(child)
	if err != nil {
		panic(err)
	}

	log.Debugf("told to accept %s as child for deployment %s", child.Id, deploymentId)

	parent := hTable.getParent(deploymentId)

	depClient := deployer.NewDeployerClient(child.Addr + ":" + strconv.Itoa(deployer.Port))
	status := depClient.WarnThatIAmParent(deploymentId, myself, parent)
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

	reqBody := api.IAmYourParentRequestBody{}
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		panic(err)
	}

	var (
		parent      *utils.Node
		grandparent *utils.Node
	)

	if len(reqBody) == 0 {
		panic("no parent in request body")
	}

	if len(reqBody) > 0 {
		parent = reqBody[api.ParentIdx]
	}

	if len(reqBody) > 1 {
		grandparent = reqBody[api.GrandparentIdx]
	}

	if parent == nil {
		panic("parent is nil")
	}

	if grandparent != nil {
		log.Debugf("told to accept %s as parent (%s grandparent) for deployment %s", parent.Id, grandparent.Id,
			deploymentId)

	} else {
		log.Debugf("told to accept %s as parent for deployment %s", parent.Id, deploymentId)
	}

	hTable.setDeploymentParent(deploymentId, parent)
	hTable.setDeploymentGrandparent(deploymentId, grandparent)
}

func getHierarchyTableHandler(w http.ResponseWriter, _ *http.Request) {
	utils.SendJSONReplyOK(w, hTable.toDTO())
}

func parentAliveHandler(_ http.ResponseWriter, r *http.Request) {
	parentId := utils.ExtractPathVar(r, nodeIdPathVar)
	log.Debugf("parent %s is alive", parentId)
	pTable.setParentUp(parentId)
}
