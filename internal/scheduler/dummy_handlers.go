package scheduler

import (
	"encoding/json"
	"net/http"
	"os"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/scheduler"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	log "github.com/sirupsen/logrus"
)

var (
	hostname string
)

func init() {
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		panic(err)
	}
}

func dummyStartInstanceHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling start instance")
	var containerInstance api.ContainerInstanceDTO
	err := json.NewDecoder(r.Body).Decode(&containerInstance)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if containerInstance.DeploymentName == "" || containerInstance.ImageName == "" {
		log.Errorf("invalid container instance: %v", containerInstance)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Debugf("[DUMMY] starting random instance for deployment %s", containerInstance.DeploymentName)

	instanceId := containerInstance.DeploymentName + "-" + utils.RandomString(10)

	portBindings := generatePortBindings(containerInstance.Ports)

	status := deployerClient.RegisterDeploymentInstance(containerInstance.DeploymentName, instanceId,
		containerInstance.Static,
		portBindings, true)

	if status != http.StatusOK {
		log.Fatalf("got status code %d while adding instances to archimedes", status)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, hostPorts := range portBindings {
		auxHostPorts := hostPorts
		go func() {
			addr := hostname + ":" + auxHostPorts[0].HostPort
			log.Debugf("listening on %s for instance %s", addr, instanceId)
			mux := http.NewServeMux()
			mux.HandleFunc("/", sendOk)
			s := &http.Server{
				Addr:    addr,
				Handler: mux,
			}
			httpErr := s.ListenAndServe()
			if httpErr != nil {
				panic(err)
			}
		}()
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

func sendOk(_ http.ResponseWriter, _ *http.Request) {
	return
}
