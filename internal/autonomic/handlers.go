package autonomic

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/goccy/go-json"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/deployment"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/scheduler"
	cedUtils "github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/golang/geo/s2"
	log "github.com/sirupsen/logrus"
)

var (
	autonomicSystem *system
	location        s2.CellID
)

func InitServer(autoFactory autonomic.ClientFactory, archFactory archimedes.ClientFactory,
	deplFactory deployer.ClientFactory, schedFactory scheduler.ClientFactory) {
	log.SetLevel(log.DebugLevel)

	locationToken, ok := os.LookupEnv(cedUtils.LocationEnvVarName)
	if !ok {
		log.Panic("location env var not set")
	}

	location = s2.CellIDFromToken(locationToken)

	autonomicSystem = newSystem(autoFactory, archFactory, deplFactory, schedFactory)

	log.SetLevel(log.InfoLevel)

	go func() {
		time.Sleep(10 * time.Second)
		autonomicSystem.start()
		deplClient := deplFactory.New()
		deplClient.SetReady(deployment.Myself.Addr + ":" + strconv.Itoa(deployer.Port))
	}()
}

func getIDHandler(w http.ResponseWriter, _ *http.Request) {
	utils.SendJSONReplyOK(w, deployment.Myself.ID)
}

func addDeploymentHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentID := utils.ExtractPathVar(r, deploymentIDPathVar)

	var deploymentConfig api.AddDeploymentRequestBody

	err := json.NewDecoder(r.Body).Decode(&deploymentConfig)
	if err != nil {
		log.Panic(err)
	}

	autonomicSystem.addDeployment(deploymentID, deploymentConfig.StrategyID, deploymentConfig.DepthFactor,
		deploymentConfig.ExploringTTL)
}

func removeDeploymentHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentID := utils.ExtractPathVar(r, deploymentIDPathVar)
	autonomicSystem.removeDeployment(deploymentID)
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

	reqBody := api.AddDeploymentChildRequestBody{}

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		log.Panic(err)
	}

	child := &reqBody
	log.Debugf("got request for adding child %s", child.ID)

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
		log.Panic(err)
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
		log.Panic(err)
	}

	closest := autonomicSystem.closestNodeTo(reqBody.Locations, reqBody.ToExclude)
	if closest == nil {
		utils.SendJSONReplyStatus(w, http.StatusNotFound, closest)

		return
	}

	utils.SendJSONReplyOK(w, closest)
}

func getMyLocationHandler(w http.ResponseWriter, _ *http.Request) {
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

	log.Debugf("deployment %s has load %d", deploymentID, load)

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
		log.Panic(err)
	}

	value, ok := autonomicSystem.deployments.Load(deploymentID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	depl := value.(deploymentsMapValue)

	log.Debugf("%s told me to blacklist %+v (%s)", reqBody.Origin, reqBody.Nodes, deploymentID)
	depl.BlacklistNodes(reqBody.Origin, reqBody.Nodes, reqBody.NodesVisited)
}
