package deployer

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	log "github.com/sirupsen/logrus"
)

func registerDeploymentInstanceHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling request in registerDeploymentInstance handler")

	deploymentID := utils.ExtractPathVar(r, deploymentIDPathVar)

	ok := hTable.hasDeployment(deploymentID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	instanceID := utils.ExtractPathVar(r, instanceIDPathVar)

	instanceDTO := deployer.RegisterDeploymentInstanceRequestBody{}

	err := json.NewDecoder(r.Body).Decode(&instanceDTO)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	if !instanceDTO.Static {
		initChan := make(chan struct{})
		initChansMap.Store(instanceID, initChan)

		go cleanUnresponsiveInstance(deploymentID, instanceID, &instanceDTO, initChan)
	} else {
		status := archimedesClient.RegisterDeploymentInstance(deploymentID, instanceID, instanceDTO.Static,
			instanceDTO.PortTranslation, instanceDTO.Local)
		if status != http.StatusOK {
			log.Debugf("got status %d while adding instance %s to archimedes", status, instanceID)
			w.WriteHeader(status)

			return
		}
		log.Debugf("warned archimedes that instance %s from deployment %s exists", instanceID, deploymentID)
	}
}

func registerHeartbeatDeploymentInstanceHandler(w http.ResponseWriter, r *http.Request) {
	deploymentID := utils.ExtractPathVar(r, deploymentIDPathVar)
	instanceID := utils.ExtractPathVar(r, instanceIDPathVar)

	pairDeploymentStatus := &pairDeploymentIDStatus{
		DeploymentID: deploymentID,
		IsUp:         true,
		Mutex:        &sync.Mutex{},
	}

	_, loaded := heartbeatsMap.LoadOrStore(instanceID, pairDeploymentStatus)
	if loaded {
		w.WriteHeader(http.StatusConflict)

		return
	}

	value, initChanOk := initChansMap.Load(instanceID)
	if !initChanOk {
		log.Warnf("ignoring heartbeat from instance %s since it didnt have an init channel", instanceID)
		w.WriteHeader(http.StatusNotFound)

		return
	}

	initChan := value.(typeInitChansMapValue)
	close(initChan)

	log.Debugf("registered deployment %s instance %s first heartbeat", deploymentID, instanceID)
}

func heartbeatDeploymentInstanceHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling request in heartbeatDeployment handler")

	deploymentID := utils.ExtractPathVar(r, deploymentIDPathVar)

	hTable.hasDeployment(deploymentID)

	instanceID := utils.ExtractPathVar(r, instanceIDPathVar)

	value, ok := heartbeatsMap.Load(instanceID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	pairDeploymentStatus := value.(typeHeartbeatsMapValue)
	pairDeploymentStatus.Mutex.Lock()
	pairDeploymentStatus.IsUp = true
	pairDeploymentStatus.Mutex.Unlock()

	log.Debugf("got heartbeat from instance %s", instanceID)
}
