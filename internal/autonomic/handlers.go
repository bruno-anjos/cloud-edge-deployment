package autonomic

import (
	"encoding/json"
	"net/http"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

var (
	autonomicSystem *System
)

func init() {
	autonomicSystem = NewAutonomicSystem()
	autonomicSystem.Start()
}

func addServiceHandler(_ http.ResponseWriter, r *http.Request) {
	serviceId := utils.ExtractPathVar(r, ServiceIdPathVar)

	var serviceConfig api.AddServiceRequestBody
	err := json.NewDecoder(r.Body).Decode(&serviceConfig)
	if err != nil {
		panic(err)
	}

	err = autonomicSystem.AddService(serviceId, serviceConfig.StrategyId)
	if err != nil {
		panic(err)
	}

	return
}

func removeServiceHandler(_ http.ResponseWriter, r *http.Request) {
	serviceId := utils.ExtractPathVar(r, ServiceIdPathVar)
	autonomicSystem.RemoveService(serviceId)
}

func getAllServicesHandler(w http.ResponseWriter, _ *http.Request) {
	var resp api.GetAllServicesResponseBody
	resp = autonomicSystem.GetServices()

	utils.SendJSONReplyOK(w, resp)
}

func addServiceChildHandler(_ http.ResponseWriter, r *http.Request) {
	serviceId := utils.ExtractPathVar(r, ServiceIdPathVar)
	childId := utils.ExtractPathVar(r, ChildIdPathVar)

	autonomicSystem.AddServiceChild(serviceId, childId)
}

func removeServiceChildHandler(_ http.ResponseWriter, r *http.Request) {
	serviceId := utils.ExtractPathVar(r, ServiceIdPathVar)
	childId := utils.ExtractPathVar(r, ChildIdPathVar)

	autonomicSystem.RemoveServiceChild(serviceId, childId)
}
