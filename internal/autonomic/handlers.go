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
	serviceId := utils.ExtractPathVar(r, ServiceIdPathVar)

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
	serviceId := utils.ExtractPathVar(r, ServiceIdPathVar)
	autonomicSystem.removeService(serviceId)
}

func getAllServicesHandler(w http.ResponseWriter, _ *http.Request) {
	var resp api.GetAllServicesResponseBody
	resp = autonomicSystem.getServices()

	utils.SendJSONReplyOK(w, resp)
}

func addServiceChildHandler(_ http.ResponseWriter, r *http.Request) {
	serviceId := utils.ExtractPathVar(r, ServiceIdPathVar)
	childId := utils.ExtractPathVar(r, ChildIdPathVar)

	autonomicSystem.addServiceChild(serviceId, childId)
}

func removeServiceChildHandler(_ http.ResponseWriter, r *http.Request) {
	serviceId := utils.ExtractPathVar(r, ServiceIdPathVar)
	childId := utils.ExtractPathVar(r, ChildIdPathVar)

	autonomicSystem.removeServiceChild(serviceId, childId)
}
