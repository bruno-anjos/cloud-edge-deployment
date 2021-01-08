package deployer

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	archimedes2 "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer/client"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/servers"
	"github.com/docker/go-connections/nat"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
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

		status := archimedesClient.RegisterDeploymentInstance(servers.ArchimedesLocalHostPort, deploymentID,
			instanceID, instanceDTO.Static,
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
	deploymentYAML := api.DeploymentYAML{}

	err := yaml.Unmarshal(hTable.getDeploymentConfig(deploymentID), &deploymentYAML)
	if err != nil {
		log.Panic(err)
	}

	instance, status := archimedesClient.GetInstance(servers.ArchimedesLocalHostPort, instanceID)
	if status != http.StatusOK {
		log.Panicf("status %d while getting instance %s", status, instanceID)
	}

	var (
		outport string
		ports   []nat.PortBinding
	)

	for _, ports = range instance.PortTranslation {
		outport = ports[0].HostPort
	}

	status = schedulerClient.StopInstance(servers.SchedulerLocalHostPort, instanceID, fmt.Sprintf("%s:%s", nodeIP,
		outport), deploymentYAML.RemovePath)
	if status != http.StatusOK {
		log.Errorf("while trying to remove instance %s after timeout, scheduler returned status %d",
			instanceID, status)
	}

	status = archimedesClient.DeleteDeploymentInstance(servers.ArchimedesLocalHostPort, deploymentID, instanceID)
	if existed {
		if status != http.StatusOK {
			log.Errorf("while trying to remove instance %s after timeout, archimedes returned status %d",
				instanceID, status)
		}
	}

	log.Errorf("Removed unresponsive instance %s", instanceID)
}
