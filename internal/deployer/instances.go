package deployer

import (
	"net/http"
	"sync"
	"time"

	archimedes2 "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	log "github.com/sirupsen/logrus"
)

type (
	typeHeartbeatsMapKey   = string
	typeHeartbeatsMapValue = *PairServiceIdStatus

	typeInitChansMapValue = chan struct{}
)

const (
	initInstanceTimeout = 30 * time.Second
)

var (
	heartbeatsMap sync.Map
	initChansMap  sync.Map
)

func init() {
	heartbeatsMap = sync.Map{}
	initChansMap = sync.Map{}

	go instanceHeartbeatChecker()
}

func cleanUnresponsiveInstance(serviceId, instanceId string, instanceDTO *archimedes2.InstanceDTO,
	alive <-chan struct{}) {
	unresponsiveTimer := time.NewTimer(initInstanceTimeout)

	select {
	case <-alive:
		log.Debugf("instance %s is up", instanceId)
		status := archimedesClient.RegisterServiceInstance(serviceId, instanceId, instanceDTO.Static,
			instanceDTO.PortTranslation, instanceDTO.Local)
		if status != http.StatusOK {
			log.Errorf("got status %d while registering service %s instance %s", status, serviceId, instanceId)
		}

		return
	case <-unresponsiveTimer.C:
		removeInstance(serviceId, instanceId)
	}
}

func instanceHeartbeatChecker() {
	heartbeatTimer := time.NewTimer(deployer.HeartbeatCheckerTimeout * time.Second)

	var toDelete []string
	for {
		toDelete = []string{}
		<-heartbeatTimer.C
		log.Debug("checking heartbeats")
		heartbeatsMap.Range(func(key, value interface{}) bool {
			instanceId := key.(typeHeartbeatsMapKey)
			pairServiceStatus := value.(typeHeartbeatsMapValue)
			pairServiceStatus.Mutex.Lock()

			// case where instance didnt set online status since last status reset, so it has to be removed
			if !pairServiceStatus.IsUp {
				pairServiceStatus.Mutex.Unlock()
				removeInstance(pairServiceStatus.ServiceId, instanceId)

				toDelete = append(toDelete, instanceId)
				log.Debugf("removing instance %s", instanceId)
			} else {
				pairServiceStatus.IsUp = false
				pairServiceStatus.Mutex.Unlock()
			}

			return true
		})

		for _, instanceId := range toDelete {
			log.Debugf("removing %s instance from expected hearbeats map", instanceId)
			heartbeatsMap.Delete(instanceId)
		}
		heartbeatTimer.Reset(deployer.HeartbeatCheckerTimeout * time.Second)
	}
}

func removeInstance(serviceId, instanceId string) {
	status := schedulerClient.StopInstance(instanceId)
	if status != http.StatusOK {
		log.Warnf("while trying to remove instance %s after timeout, scheduler returned status %d",
			instanceId, status)
	}

	status = archimedesClient.DeleteServiceInstance(serviceId, instanceId)
	if status != http.StatusOK {
		log.Warnf("while trying to remove instance %s after timeout, archimedes returned status %d",
			instanceId, status)
	}
}
