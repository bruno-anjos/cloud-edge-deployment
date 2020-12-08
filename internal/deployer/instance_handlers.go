package deployer

import (
	"encoding/json"
	"net/http"
	"sync"

	archimedes2 "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	log "github.com/sirupsen/logrus"
)

func registerDeploymentInstanceHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling request in registerDeploymentInstance handler")

	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)

	ok := hTable.hasDeployment(deploymentId)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	instanceId := utils.ExtractPathVar(r, instanceIdPathVar)

	instanceDTO := archimedes2.InstanceDTO{}
	err := json.NewDecoder(r.Body).Decode(&instanceDTO)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !instanceDTO.Static {
		initChan := make(chan struct{})
		initChansMap.Store(instanceId, initChan)
		go cleanUnresponsiveInstance(deploymentId, instanceId, &instanceDTO, initChan)
	} else {
		status := archimedesClient.RegisterDeploymentInstance(deploymentId, instanceId, instanceDTO.Static,
			instanceDTO.PortTranslation, instanceDTO.Local)
		if status != http.StatusOK {
			log.Debugf("got status %d while adding instance %s to archimedes", status, instanceId)
			w.WriteHeader(status)
			return
		}
		log.Debugf("warned archimedes that instance %s from deployment %s exists", instanceId, deploymentId)
	}
}

func registerHeartbeatDeploymentInstanceHandler(w http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)
	instanceId := utils.ExtractPathVar(r, instanceIdPathVar)

	pairDeploymentStatus := &pairDeploymentIdStatus{
		DeploymentId: deploymentId,
		IsUp:         true,
		Mutex:        &sync.Mutex{},
	}

	_, loaded := heartbeatsMap.LoadOrStore(instanceId, pairDeploymentStatus)
	if loaded {
		w.WriteHeader(http.StatusConflict)
		return
	}

	value, initChanOk := initChansMap.Load(instanceId)
	if !initChanOk {
		log.Warnf("ignoring heartbeat from instance %s since it didnt have an init channel", instanceId)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	initChan := value.(typeInitChansMapValue)
	close(initChan)

	log.Debugf("registered deployment %s instance %s first heartbeat", deploymentId, instanceId)
}

func heartbeatDeploymentInstanceHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling request in heartbeatDeployment handler")

	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)

	hTable.hasDeployment(deploymentId)

	instanceId := utils.ExtractPathVar(r, instanceIdPathVar)

	value, ok := heartbeatsMap.Load(instanceId)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	pairDeploymentStatus := value.(typeHeartbeatsMapValue)
	pairDeploymentStatus.Mutex.Lock()
	pairDeploymentStatus.IsUp = true
	pairDeploymentStatus.Mutex.Unlock()

	log.Debugf("got heartbeat from instance %s", instanceId)
}
