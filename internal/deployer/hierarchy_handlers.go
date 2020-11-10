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

	go attemptToExtend(deploymentId, "", config, 0, body.Alternatives, api.NotExploringTTL)
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

	go attemptToExtend(deploymentId, reqBody.OrphanId, config, maxHopsToLookFor, nil, api.NotExploringTTL)
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

func iAmYourParentHandler(w http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)

	reqBody := api.IAmYourParentRequestBody{}
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		panic(err)
	}

	parentId := "nil"
	if reqBody.Parent != nil {
		parentId = reqBody.Parent.Id
	}

	grandparentId := "nil"
	if reqBody.Grandparent != nil {
		grandparentId = reqBody.Grandparent.Id
	}

	log.Debugf("told to accept %s as parent and %s as grandparent for deployment %s", parentId, grandparentId,
		deploymentId)

	parent := hTable.getParent(deploymentId)
	hasParent := parent != nil
	deadParent := false
	if hasParent {
		deadParent = !pTable.hasParent(parent.Id)
	}

	if hasParent && !deadParent {
		log.Debugf("rejecting parent %s, since i have %s and he is not dead", parentId, parent.Id)
		w.WriteHeader(http.StatusConflict)
		return
	}

	hTable.setDeploymentParent(deploymentId, parent)
	hTable.setDeploymentGrandparent(deploymentId, reqBody.Grandparent)
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
