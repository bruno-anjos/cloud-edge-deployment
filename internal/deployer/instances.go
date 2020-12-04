package deployer

import (
	"net/http"
	"sync"
	"time"

	archimedes2 "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer/client"

	log "github.com/sirupsen/logrus"
)

type (
	typeHeartbeatsMapKey   = string
	typeHeartbeatsMapValue = *PairDeploymentIdStatus

	typeInitChansMapValue = chan struct{}
)

const (
	initInstanceTimeout = 60 * time.Second
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

func cleanUnresponsiveInstance(deploymentId, instanceId string, instanceDTO *archimedes2.InstanceDTO,
	alive <-chan struct{}) {
	unresponsiveTimer := time.NewTimer(initInstanceTimeout)

	select {
	case <-alive:
		log.Debugf("instance %s is up", instanceId)
		status := archimedesClient.RegisterDeploymentInstance(deploymentId, instanceId, instanceDTO.Static,
			instanceDTO.PortTranslation, instanceDTO.Local)
		if status != http.StatusOK {
			log.Errorf("got status %d while registering deployment %s instance %s", status, deploymentId, instanceId)
		}

		return
	case <-unresponsiveTimer.C:
		log.Debugf("%s for deployment %s never sent heartbeat", instanceId, deploymentId)
		removeInstance(deploymentId, instanceId, false)
	}
}

func instanceHeartbeatChecker() {
	heartbeatTimer := time.NewTimer(client.HeartbeatCheckerTimeout * time.Second)

	var toDelete []string
	for {
		toDelete = []string{}
		<-heartbeatTimer.C
		log.Debug("checking heartbeats")
		heartbeatsMap.Range(func(key, value interface{}) bool {
			instanceId := key.(typeHeartbeatsMapKey)
			pairDeploymentStatus := value.(typeHeartbeatsMapValue)
			pairDeploymentStatus.Mutex.Lock()

			// case where instance didnt set online status since last status reset, so it has to be removed
			if !pairDeploymentStatus.IsUp {
				pairDeploymentStatus.Mutex.Unlock()
				removeInstance(pairDeploymentStatus.DeploymentId, instanceId, true)

				toDelete = append(toDelete, instanceId)
				log.Debugf("removing instance %s", instanceId)
			} else {
				pairDeploymentStatus.IsUp = false
				pairDeploymentStatus.Mutex.Unlock()
			}

			return true
		})

		for _, instanceId := range toDelete {
			log.Debugf("removing %s instance from expected hearbeats map", instanceId)
			heartbeatsMap.Delete(instanceId)
		}
		heartbeatTimer.Reset(client.HeartbeatCheckerTimeout * time.Second)
	}
}

func removeInstance(deploymentId, instanceId string, existed bool) {
	status := schedulerClient.StopInstance(instanceId)
	if status != http.StatusOK {
		log.Errorf("while trying to remove instance %s after timeout, scheduler returned status %d",
			instanceId, status)
	}

	status = archimedesClient.DeleteDeploymentInstance(deploymentId, instanceId)
	if existed {
		if status != http.StatusOK {
			log.Errorf("while trying to remove instance %s after timeout, archimedes returned status %d",
				instanceId, status)
		}
	}

	log.Errorf("Removed unresponsive instance %s", instanceId)
}
