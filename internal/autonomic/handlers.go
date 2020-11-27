package autonomic

import (
	"encoding/json"
	"net/http"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	log "github.com/sirupsen/logrus"
)

var (
	autonomicSystem *system
)

func init() {
	log.SetLevel(log.DebugLevel)
	autonomicSystem = newSystem()

	log.SetLevel(log.InfoLevel)
}

func addDeploymentHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)

	var deploymentConfig api.AddDeploymentRequestBody
	err := json.NewDecoder(r.Body).Decode(&deploymentConfig)
	if err != nil {
		panic(err)
	}

	autonomicSystem.addDeployment(deploymentId, deploymentConfig.StrategyId, deploymentConfig.DepthFactor,
		deploymentConfig.ExploringTTL)

	return
}

func removeDeploymentHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)
	autonomicSystem.removeDeployment(deploymentId)
}

func getAllDeploymentsHandler(w http.ResponseWriter, _ *http.Request) {
	resp := api.GetAllDeploymentsResponseBody{}
	deployments := autonomicSystem.getDeployments()
	for deploymentId, s := range deployments {
		resp[deploymentId] = s.ToDTO()
	}

	utils.SendJSONReplyOK(w, resp)
}

func addDeploymentChildHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)
	// TODO missing child in body

	reqBody := api.AddDeploymentChildRequestBody{}
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		panic(err)
	}

	child := &reqBody
	autonomicSystem.addDeploymentChild(deploymentId, child)
}

func removeDeploymentChildHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)
	childId := utils.ExtractPathVar(r, childIdPathVar)

	autonomicSystem.removeDeploymentChild(deploymentId, childId)
}

func setDeploymentParentHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)

	reqBody := api.SetDeploymentParentRequestBody{}
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		panic(err)
	}

	parent := &reqBody

	log.Debugf("setting %s as parent for deployment %s", parent.Id, deploymentId)
	autonomicSystem.setDeploymentParent(deploymentId, parent)
}

func isNodeInVicinityHandler(w http.ResponseWriter, r *http.Request) {
	nodeId := utils.ExtractPathVar(r, nodeIdPathVar)

	if !autonomicSystem.isNodeInVicinity(nodeId) {
		w.WriteHeader(http.StatusNotFound)
	}

	return
}

func closestNodeToHandler(w http.ResponseWriter, r *http.Request) {
	var reqBody api.ClosestNodeRequestBody
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		panic(err)
	}

	closest := autonomicSystem.closestNodeTo(reqBody.Locations, reqBody.ToExclude)
	if closest == nil {
		utils.SendJSONReplyStatus(w, http.StatusNotFound, closest)
		return
	}

	utils.SendJSONReplyOK(w, closest)
}

func getVicinityHandler(w http.ResponseWriter, _ *http.Request) {
	vicinity := autonomicSystem.getVicinity()
	if vicinity == nil {
		utils.SendJSONReplyStatus(w, http.StatusNotFound, nil)
		return
	}

	var respBody api.GetVicinityResponseBody
	respBody = *vicinity

	utils.SendJSONReplyOK(w, respBody)
}

func getMyLocationHandler(w http.ResponseWriter, _ *http.Request) {
	location, err := autonomicSystem.getMyLocation()
	if err != nil {
		utils.SendJSONReplyStatus(w, http.StatusNotFound, 0)
		return
	}

	var respBody api.GetMyLocationResponseBody
	respBody = location

	utils.SendJSONReplyOK(w, respBody)
}

func getLoadForDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)
	load, ok := autonomicSystem.getLoad(deploymentId)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	log.Debugf("deployment %s has load %f", deploymentId, load)

	utils.SendJSONReplyOK(w, load)
}

func setExploreSuccessfullyHandler(w http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)
	childId := utils.ExtractPathVar(r, childIdPathVar)

	_, ok := autonomicSystem.deployments.Load(deploymentId)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	log.Debugf("setting explore success %s %s", deploymentId, childId)

	ok = autonomicSystem.setExploreSuccess(deploymentId, childId)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	log.Debugf("explored deployment %s through %s successfully", deploymentId, childId)
}

func blacklistNodeHandler(w http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)

	var reqBody api.BlacklistNodeRequestBody
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		panic(err)
	}

	value, ok := autonomicSystem.deployments.Load(deploymentId)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	depl := value.(deploymentsMapValue)

	log.Debugf("%s told me to blacklist %+v", reqBody.Origin, reqBody.Nodes)
	depl.BlacklistNodes(reqBody.Origin, reqBody.Nodes...)
}
