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
	typeHeartbeatsMapValue = *pairDeploymentIDStatus

	typeInitChansMapValue = chan struct{}
)

const (
	initInstanceTimeout = 60 * time.Second
)

var (
	heartbeatsMap = sync.Map{}
	initChansMap  = sync.Map{}
)

func cleanUnresponsiveInstance(deploymentID, instanceID string, instanceDTO *archimedes2.InstanceDTO,
	alive <-chan struct{}) {
	unresponsiveTimer := time.NewTimer(initInstanceTimeout)

	select {
	case <-alive:
		log.Debugf("instance %s is up", instanceID)

		status := archimedesClient.RegisterDeploymentInstance(deploymentID, instanceID, instanceDTO.Static,
			instanceDTO.PortTranslation, instanceDTO.Local)
		if status != http.StatusOK {
			log.Errorf("got status %d while registering deployment %s instance %s", status, deploymentID, instanceID)
		}

		return
	case <-unresponsiveTimer.C:
		log.Debugf("%s for deployment %s never sent heartbeat", instanceID, deploymentID)
		removeInstance(deploymentID, instanceID, false)
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
			instanceID := key.(typeHeartbeatsMapKey)
			pairDeploymentStatus := value.(typeHeartbeatsMapValue)
			pairDeploymentStatus.Mutex.Lock()

			// case where instance didnt set online status since last status reset, so it has to be removed
			if !pairDeploymentStatus.IsUp {
				pairDeploymentStatus.Mutex.Unlock()
				removeInstance(pairDeploymentStatus.DeploymentID, instanceID, true)

				toDelete = append(toDelete, instanceID)
				log.Debugf("removing instance %s", instanceID)
			} else {
				pairDeploymentStatus.IsUp = false
				pairDeploymentStatus.Mutex.Unlock()
			}

			return true
		})

		for _, instanceID := range toDelete {
			log.Debugf("removing %s instance from expected hearbeats map", instanceID)
			heartbeatsMap.Delete(instanceID)
		}

		heartbeatTimer.Reset(client.HeartbeatCheckerTimeout * time.Second)
	}
}

func removeInstance(deploymentID, instanceID string, existed bool) {
	status := schedulerClient.StopInstance(instanceID)
	if status != http.StatusOK {
		log.Errorf("while trying to remove instance %s after timeout, scheduler returned status %d",
			instanceID, status)
	}

	status = archimedesClient.DeleteDeploymentInstance(deploymentID, instanceID)
	if existed {
		if status != http.StatusOK {
			log.Errorf("while trying to remove instance %s after timeout, archimedes returned status %d",
				instanceID, status)
		}
	}

	heartbeatsMap.Delete(instanceID)

	log.Errorf("Removed unresponsive instance %s", instanceID)
}
