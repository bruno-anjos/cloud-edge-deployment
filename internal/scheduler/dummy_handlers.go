package scheduler

import (
	"net/http"

	"github.com/goccy/go-json"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/scheduler"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/servers"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	log "github.com/sirupsen/logrus"
)

const randomChars = 10

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

	instanceID := containerInstance.DeploymentName + "-" + servers.RandomString(randomChars)

	portBindings := generatePortBindings(containerInstance.Ports)

	status := deplClient.RegisterDeploymentInstance(servers.DeployerLocalHostPort, containerInstance.DeploymentName,
		instanceID,
		containerInstance.Static,
		portBindings, true)

	if status != http.StatusOK {
		log.Panicf("got status code %d while adding instances to archimedes", status)
		w.WriteHeader(http.StatusInternalServerError)

		return //nolint:wsl
	}

	for _, hostPorts := range portBindings {
		auxHostPorts := hostPorts

		go func() {
			addr := myself.Addr + ":" + auxHostPorts[0].HostPort
			log.Debugf("listening on %s for instance %s", addr, instanceID)

			mux := http.NewServeMux()
			mux.HandleFunc("/", sendOk)

			s := &http.Server{
				Addr:              addr,
				Handler:           mux,
				TLSConfig:         nil,
				ReadTimeout:       0,
				ReadHeaderTimeout: 0,
				WriteTimeout:      0,
				IdleTimeout:       0,
				MaxHeaderBytes:    0,
				TLSNextProto:      nil,
				ConnState:         nil,
				ErrorLog:          nil,
				BaseContext:       nil,
				ConnContext:       nil,
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

	instanceID := utils.ExtractPathVar(r, instanceIDPathVar)

	if instanceID == "" {
		log.Error("no instance provided")
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	log.Debugf("[DUMMY] stopping instance %s", instanceID)
}

func dummyStopAllInstancesHandler(_ http.ResponseWriter, _ *http.Request) {
	log.Debug("[DUMMY] stopping all instances")
}

func sendOk(_ http.ResponseWriter, _ *http.Request) {}
