package deployer

import (
	"encoding/json"
	"net/http"
	"sync"

	archimedes2 "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	log "github.com/sirupsen/logrus"
)

func registerServiceInstanceHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling request in registerServiceInstance handler")

	deploymentId := utils.ExtractPathVar(r, DeploymentIdPathVar)

	ok := hierarchyTable.HasDeployment(deploymentId)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	instanceId := utils.ExtractPathVar(r, InstanceIdPathVar)

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
		status := archimedesClient.RegisterServiceInstance(deploymentId, instanceId, instanceDTO.Static,
			instanceDTO.PortTranslation, instanceDTO.Local)
		if status != http.StatusOK {
			log.Debugf("got status %d while adding instance %s to archimedes", status, instanceId)
			w.WriteHeader(status)
			return
		}
		log.Debugf("warned archimedes that instance %s from service %s exists", instanceId, deploymentId)
	}
}

func registerHeartbeatServiceInstanceHandler(w http.ResponseWriter, r *http.Request) {
	serviceId := utils.ExtractPathVar(r, DeploymentIdPathVar)
	instanceId := utils.ExtractPathVar(r, InstanceIdPathVar)

	pairServiceStatus := &PairServiceIdStatus{
		ServiceId: serviceId,
		IsUp:      true,
		Mutex:     &sync.Mutex{},
	}

	_, loaded := heartbeatsMap.LoadOrStore(instanceId, pairServiceStatus)
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

	log.Debugf("registered service %s instance %s first heartbeat", serviceId, instanceId)
}

func heartbeatServiceInstanceHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling request in heartbeatService handler")

	deploymentId := utils.ExtractPathVar(r, DeploymentIdPathVar)

	hierarchyTable.HasDeployment(deploymentId)

	instanceId := utils.ExtractPathVar(r, InstanceIdPathVar)

	value, ok := heartbeatsMap.Load(instanceId)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	pairServiceStatus := value.(typeHeartbeatsMapValue)
	pairServiceStatus.Mutex.Lock()
	pairServiceStatus.IsUp = true
	pairServiceStatus.Mutex.Unlock()

	log.Debugf("got heartbeat from instance %s", instanceId)
}
