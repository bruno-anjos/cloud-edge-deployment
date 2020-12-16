package autonomic

import (
	"encoding/json"
	"net/http"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/scheduler"
	log "github.com/sirupsen/logrus"
)

var autonomicSystem *system

func InitServer(autoFactory autonomic.ClientFactory, archFactory archimedes.ClientFactory,
	deplFactory deployer.ClientFactory, schedFactory scheduler.ClientFactory) {
	log.SetLevel(log.DebugLevel)

	autonomicSystem = newSystem(autoFactory, archFactory, deplFactory, schedFactory)

	log.SetLevel(log.InfoLevel)
}

func addDeploymentHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentID := utils.ExtractPathVar(r, deploymentIDPathVar)

	var deploymentConfig api.AddDeploymentRequestBody

	err := json.NewDecoder(r.Body).Decode(&deploymentConfig)
	if err != nil {
		panic(err)
	}

	autonomicSystem.addDeployment(deploymentID, deploymentConfig.StrategyID, deploymentConfig.DepthFactor,
		deploymentConfig.ExploringTTL)
}

func removeDeploymentHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentID := utils.ExtractPathVar(r, deploymentIDPathVar)
	autonomicSystem.removeDeployment(deploymentID, "a")
}

func getAllDeploymentsHandler(w http.ResponseWriter, _ *http.Request) {
	resp := api.GetAllDeploymentsResponseBody{}
	deployments := autonomicSystem.getDeployments()

	for deploymentID, s := range deployments {
		resp[deploymentID] = s.ToDTO()
	}

	utils.SendJSONReplyOK(w, resp)
}

func addDeploymentChildHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentID := utils.ExtractPathVar(r, deploymentIDPathVar)
	// TODO missing child in body

	reqBody := api.AddDeploymentChildRequestBody{}

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		panic(err)
	}

	child := &reqBody
	autonomicSystem.addDeploymentChild(deploymentID, child)
}

func removeDeploymentChildHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentID := utils.ExtractPathVar(r, deploymentIDPathVar)
	childID := utils.ExtractPathVar(r, childIDPathVar)

	autonomicSystem.removeDeploymentChild(deploymentID, childID)
}

func setDeploymentParentHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentID := utils.ExtractPathVar(r, deploymentIDPathVar)

	reqBody := api.SetDeploymentParentRequestBody{}

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		panic(err)
	}

	parent := &reqBody

	log.Debugf("setting %s as parent for deployment %s", parent.ID, deploymentID)
	autonomicSystem.setDeploymentParent(deploymentID, parent)
}

func isNodeInVicinityHandler(w http.ResponseWriter, r *http.Request) {
	nodeID := utils.ExtractPathVar(r, nodeIDPathVar)

	if !autonomicSystem.isNodeInVicinity(nodeID) {
		w.WriteHeader(http.StatusNotFound)
	}
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

	respBody := *vicinity

	utils.SendJSONReplyOK(w, respBody)
}

func getMyLocationHandler(w http.ResponseWriter, _ *http.Request) {
	location, err := autonomicSystem.getMyLocation()
	if err != nil {
		utils.SendJSONReplyStatus(w, http.StatusNotFound, 0)

		return
	}

	respBody := location

	utils.SendJSONReplyOK(w, respBody)
}

func getLoadForDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	deploymentID := utils.ExtractPathVar(r, deploymentIDPathVar)

	load, ok := autonomicSystem.getLoad(deploymentID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	log.Debugf("deployment %s has load %f", deploymentID, load)

	utils.SendJSONReplyOK(w, load)
}

func setExploreSuccessfullyHandler(w http.ResponseWriter, r *http.Request) {
	deploymentID := utils.ExtractPathVar(r, deploymentIDPathVar)
	childID := utils.ExtractPathVar(r, childIDPathVar)

	_, ok := autonomicSystem.deployments.Load(deploymentID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	log.Debugf("setting explore success %s %s", deploymentID, childID)

	ok = autonomicSystem.setExploreSuccess(deploymentID, childID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	log.Debugf("explored deployment %s through %s successfully", deploymentID, childID)
}

func blacklistNodeHandler(w http.ResponseWriter, r *http.Request) {
	deploymentID := utils.ExtractPathVar(r, deploymentIDPathVar)

	var reqBody api.BlacklistNodeRequestBody

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		panic(err)
	}

	value, ok := autonomicSystem.deployments.Load(deploymentID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	depl := value.(deploymentsMapValue)

	log.Debugf("%s told me to blacklist %+v", reqBody.Origin, reqBody.Nodes)
	depl.BlacklistNodes(reqBody.Origin, reqBody.Nodes...)
}
