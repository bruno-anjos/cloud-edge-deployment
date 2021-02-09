package deployer

import (
	"net/http"

	"github.com/goccy/go-json"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	internalUtils "github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/golang/geo/s2"
	log "github.com/sirupsen/logrus"
)

func deadChildHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentID := internalUtils.ExtractPathVar(r, deploymentIDPathVar)
	deadChildID := internalUtils.ExtractPathVar(r, nodeIDPathVar)

	body := api.DeadChildRequestBody{}

	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		panic(err)
	}

	log.Debugf("grandchild %s reported deployment %s from %s as dead", body.Grandchild.ID, deploymentID, deadChildID)

	_, okChild := suspectedChild.Load(deadChildID)
	if !okChild {
		suspectedChild.Store(deadChildID, nil)
		children.Delete(deadChildID)
	}

	_, okDeployment := suspectedDeployments.Load(deploymentID)
	if !okDeployment {
		suspectedDeployments.Store(deploymentID, nil)
	}

	if !okChild || !okDeployment {
		hTable.removeChild(deploymentID, deadChildID)
	}

	config := &api.ExtendDeploymentConfig{
		Children:  []*utils.Node{body.Grandchild},
		Locations: body.Locations,
		ToExclude: nil,
	}

	go attemptToExtend(deploymentID, nil, config, 0, body.Alternatives, api.NotExploringTTL)
}

func fallbackHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentID := internalUtils.ExtractPathVar(r, deploymentIDPathVar)

	reqBody := api.FallbackRequestBody{}

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		panic(err)
	}

	log.Debugf("node %s is falling back from %s with deployment %s", reqBody.Orphan.ID,
		reqBody.OrphanLocation.ToToken(), deploymentID)

	config := &api.ExtendDeploymentConfig{
		Children:  nil,
		Locations: []s2.CellID{reqBody.OrphanLocation},
		ToExclude: nil,
	}

	go attemptToExtend(deploymentID, reqBody.Orphan, config, maxHopsToLookFor, nil, api.NotExploringTTL)
}

func setGrandparentHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentID := internalUtils.ExtractPathVar(r, deploymentIDPathVar)

	reqBody := api.SetGrandparentRequestBody{}

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		panic(err)
	}

	grandparent := &reqBody

	log.Debugf("setting %s as grandparent", grandparent.ID)

	hTable.setDeploymentGrandparent(deploymentID, grandparent)
}

func iAmYourParentHandler(w http.ResponseWriter, r *http.Request) {
	deploymentID := internalUtils.ExtractPathVar(r, deploymentIDPathVar)

	reqBody := api.IAmYourParentRequestBody{}

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		panic(err)
	}

	parentID := "nil"
	if reqBody.Parent != nil {
		parentID = reqBody.Parent.ID
	}

	grandparentID := "nil"
	if reqBody.Grandparent != nil {
		grandparentID = reqBody.Grandparent.ID
	}

	log.Debugf("told to accept %s as parent and %s as grandparent for deployment %s", parentID, grandparentID,
		deploymentID)

	parent := hTable.getParent(deploymentID)
	hasParent := parent != nil
	deadParent := false

	if hasParent {
		deadParent = !pTable.hasParent(parent.ID)
	}

	if hasParent && !deadParent {
		log.Debugf("rejecting parent %s, since i have %s and he is not dead", parentID, parent.ID)
		w.WriteHeader(http.StatusConflict)

		return
	}

	hTable.setDeploymentParent(deploymentID, parent)
	hTable.setDeploymentGrandparent(deploymentID, reqBody.Grandparent)
}

func getHierarchyTableHandler(w http.ResponseWriter, _ *http.Request) {
	internalUtils.SendJSONReplyOK(w, hTable.toDTO())
}

func parentAliveHandler(_ http.ResponseWriter, r *http.Request) {
	parentID := internalUtils.ExtractPathVar(r, nodeIDPathVar)
	log.Debugf("parent %s is alive", parentID)
	pTable.setParentUp(parentID)
}
