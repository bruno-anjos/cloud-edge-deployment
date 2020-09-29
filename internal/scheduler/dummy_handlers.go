package scheduler

import (
	"encoding/json"
	"net/http"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/scheduler"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	log "github.com/sirupsen/logrus"
)



func dummyStartInstanceHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling start instance")
	var containerInstance api.ContainerInstanceDTO
	err := json.NewDecoder(r.Body).Decode(&containerInstance)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if containerInstance.ServiceName == "" || containerInstance.ImageName == "" {
		log.Errorf("invalid container instance: %v", containerInstance)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Debugf("[DUMMY] starting random instance for service %s", containerInstance.ServiceName)

	instanceId := containerInstance.ServiceName + "-" + utils.RandomString(10)

	portBindings := generatePortBindings(containerInstance.Ports)

	status := deployerClient.RegisterServiceInstance(containerInstance.ServiceName, instanceId,
		containerInstance.Static,
		portBindings, true)

	if status != http.StatusOK {
		log.Fatalf("got status code %d while adding instances to archimedes", status)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func dummyStopInstanceHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling delete instance")

	instanceId := utils.ExtractPathVar(r, instanceIdPathVar)

	if instanceId == "" {
		log.Errorf("no instance provided", instanceId)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Debugf("[DUMMY] stopping instance %s", instanceId)
}

func dummyStopAllInstancesHandler(_ http.ResponseWriter, _ *http.Request) {
	log.Debug("[DUMMY] stopping all instances")
}
