package deployer

import (
	"encoding/json"
	"net/http"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/golang/geo/s2"
	log "github.com/sirupsen/logrus"
)

func deadChildHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)
	deadChildId := utils.ExtractPathVar(r, nodeIdPathVar)

	body := api.DeadChildRequestBody{}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		panic(err)
	}

	log.Debugf("grandchild %s reported deployment %s from %s as dead", body.Grandchild.Id, deploymentId, deadChildId)

	_, okChild := suspectedChild.Load(deadChildId)
	if !okChild {
		suspectedChild.Store(deadChildId, nil)
		children.Delete(deadChildId)
	}

	_, okDeployment := suspectedDeployments.Load(deploymentId)
	if !okDeployment {
		suspectedDeployments.Store(deploymentId, nil)
	}

	if !okChild || !okDeployment {
		hTable.removeChild(deploymentId, deadChildId)
	}

	config := &api.ExtendDeploymentConfig{
		Children:  []*utils.Node{body.Grandchild},
		Locations: body.Locations,
		ToExclude: nil,
	}

	go attemptToExtend(deploymentId, "", config, 0, body.Alternatives, false)
}

func fallbackHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)

	reqBody := api.FallbackRequestBody{}
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		panic(err)
	}

	log.Debugf("node %s is falling back from %f with deployment %s", reqBody.OrphanId, reqBody.OrphanLocation,
		deploymentId)

	config := &api.ExtendDeploymentConfig{
		Children:  nil,
		Locations: []s2.CellID{reqBody.OrphanLocation},
		ToExclude: nil,
	}

	go attemptToExtend(deploymentId, reqBody.OrphanId, config, maxHopsToLookFor, nil, false)
}

func setGrandparentHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)

	reqBody := api.SetGrandparentRequestBody{}
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		panic(err)
	}

	grandparent := &reqBody

	log.Debugf("setting %s as grandparent", grandparent.Id)

	hTable.setDeploymentGrandparent(deploymentId, grandparent)
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

func iAmYourChildHandler(w http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)

	reqBody := api.IAmYourChildRequestBody{}
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		panic(err)
	}

	if !hTable.hasDeployment(deploymentId) {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	log.Debugf("told to accept %s as child for deployment %s", reqBody.Child.Id, deploymentId)

	hTable.addChild(deploymentId, reqBody.Child)

	var responseBody *api.IAmYourChildResponseBody
	responseBody = hTable.getGrandparent(deploymentId)

	utils.SendJSONReplyOK(w, responseBody)
}

func getHierarchyTableHandler(w http.ResponseWriter, _ *http.Request) {
	utils.SendJSONReplyOK(w, hTable.toDTO())
}

func parentAliveHandler(_ http.ResponseWriter, r *http.Request) {
	parentId := utils.ExtractPathVar(r, nodeIdPathVar)
	log.Debugf("parent %s is alive", parentId)
	pTable.setParentUp(parentId)
}
