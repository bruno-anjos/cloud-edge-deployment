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
	autonomicSystem.start()

	log.SetLevel(log.InfoLevel)
}

func addServiceHandler(_ http.ResponseWriter, r *http.Request) {
	serviceId := utils.ExtractPathVar(r, serviceIdPathVar)

	var serviceConfig api.AddServiceRequestBody
	err := json.NewDecoder(r.Body).Decode(&serviceConfig)
	if err != nil {
		panic(err)
	}

	err = autonomicSystem.addService(serviceId, serviceConfig.StrategyId)
	if err != nil {
		panic(err)
	}

	return
}

func removeServiceHandler(_ http.ResponseWriter, r *http.Request) {
	serviceId := utils.ExtractPathVar(r, serviceIdPathVar)
	autonomicSystem.removeService(serviceId)
}

func getAllServicesHandler(w http.ResponseWriter, _ *http.Request) {
	resp := api.GetAllServicesResponseBody{}
	services := autonomicSystem.getServices()
	for serviceId, s := range services {
		resp[serviceId] = s.toDTO()
	}

	utils.SendJSONReplyOK(w, resp)
}

func addServiceChildHandler(_ http.ResponseWriter, r *http.Request) {
	serviceId := utils.ExtractPathVar(r, serviceIdPathVar)
	childId := utils.ExtractPathVar(r, childIdPathVar)

	autonomicSystem.addServiceChild(serviceId, childId)
}

func removeServiceChildHandler(_ http.ResponseWriter, r *http.Request) {
	serviceId := utils.ExtractPathVar(r, serviceIdPathVar)
	childId := utils.ExtractPathVar(r, childIdPathVar)

	autonomicSystem.removeServiceChild(serviceId, childId)
}

func setServiceParentHandler(_ http.ResponseWriter, r *http.Request) {
	serviceId := utils.ExtractPathVar(r, serviceIdPathVar)
	parentId := utils.ExtractPathVar(r, parentIdPathVar)

	autonomicSystem.setServiceParent(serviceId, parentId)
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

	closest := autonomicSystem.closestNodeTo(reqBody.Location, reqBody.ToExclude)
	if closest == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	utils.SendJSONReplyOK(w, closest)
}

func getVicinityHandler(w http.ResponseWriter, _ *http.Request) {
	vicinity := autonomicSystem.getVicinity()
	if vicinity == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var respBody api.GetVicinityResponseBody
	respBody = vicinity

	utils.SendJSONReplyOK(w, respBody)
}

func getMyLocationHandler(w http.ResponseWriter, _ *http.Request) {
	location := autonomicSystem.getMyLocation()
	if location == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var respBody api.GetMyLocationResponseBody
	respBody = location

	utils.SendJSONReplyOK(w, respBody)
}

func getLoadForServiceHandler(w http.ResponseWriter, r *http.Request) {
	serviceId := utils.ExtractPathVar(r, serviceIdPathVar)
	load, ok := autonomicSystem.getLoad(serviceId)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	log.Debugf("service %s has load %f", serviceId, load)

	utils.SendJSONReplyOK(w, load)
}

func setExploreSuccessfullyHandler(w http.ResponseWriter, r *http.Request) {
	serviceId := utils.ExtractPathVar(r, serviceIdPathVar)
	childId := utils.ExtractPathVar(r, childIdPathVar)

	_, ok := autonomicSystem.services.Load(serviceId)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	ok = autonomicSystem.setExploreSuccess(serviceId, childId)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	log.Debugf("explored service %s through %s successfully", serviceId, childId)
}
