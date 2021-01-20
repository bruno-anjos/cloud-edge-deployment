package archimedes

import (
	"sync"
	"time"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	log "github.com/sirupsen/logrus"
)

type (
	remoteDeploymentsTable struct {
		neighborsDeploymentsMap sync.Map
		nodeInstances           sync.Map
		staleNodes              sync.Map
		*deploymentsTable
	}

	typeNodeInstanceKey   = string
	typeNodeInstanceValue = *sync.Map
)

func newRemoteDeploymentsTable() *remoteDeploymentsTable {
	return &remoteDeploymentsTable{
		neighborsDeploymentsMap: sync.Map{},
		nodeInstances:           sync.Map{},
		staleNodes:              sync.Map{},
		deploymentsTable:        newDeploymentsTable(),
	}
}

func (r *remoteDeploymentsTable) updateFromDiscoverMsg(host *utils.Node, discMsg *api.DiscoverMsg) {
	nodeInstancesMap := &sync.Map{}

	outdatedInstances := map[string]interface{}{}
	value, ok := r.nodeInstances.Load(host.ID)
	if ok {
		nodeInstancesMap = value.(typeNodeInstanceValue)
		nodeInstancesMap.Range(func(key, value interface{}) bool {
			instanceID := key.(typeNodeInstanceKey)
			outdatedInstances[instanceID] = nil

			return true
		})
	}

	value, ok = r.nodeInstances.LoadOrStore(host.ID, nodeInstancesMap)
	if ok {
		nodeInstancesMap = value.(typeNodeInstanceValue)
	}

	for deploymentID, entry := range discMsg.Entries {
		currEntry := entry
		currEntry.MaxHops = 0

		added := r.addDeployment(deploymentID, currEntry)
		if !added {
			for instanceID, instance := range entry.Instances {
				r.addInstance(deploymentID, instanceID, instance)
				log.Debugf("added instance %s from %s: %+v", instanceID, host.ID, instance)

				delete(outdatedInstances, instanceID)
				nodeInstancesMap.Store(instanceID, instance)
			}
		}
	}

	for instanceID := range outdatedInstances {
		log.Debugf("deleting outdated instance %s from %s", instanceID, host.ID)
		r.deleteInstance(instanceID)
	}

	r.staleNodes.Delete(host.ID)
}

func (r *remoteDeploymentsTable) removeInstancesFromNode(nodeID string) {
	value, ok := r.nodeInstances.Load(nodeID)
	if !ok {
		return
	}

	nodeInstancesMap := value.(typeNodeInstanceValue)
	nodeInstancesMap.Range(func(key, value interface{}) bool {
		instanceID := key.(string)
		r.deleteInstance(instanceID)
		return true
	})
}

func (r *remoteDeploymentsTable) cleanStaleNodes() {
	staleNodesTimeout := 15 * time.Second
	timer := time.NewTimer(staleNodesTimeout)

	for {
		<-timer.C

		r.staleNodes.Range(func(key, value interface{}) bool {
			nodeID := key.(string)
			log.Debugf("removing stale node %s", nodeID)
			r.removeInstancesFromNode(nodeID)

			r.staleNodes.Delete(nodeID)

			return true
		})

		r.nodeInstances.Range(func(key, value interface{}) bool {
			nodeID := key.(typeNodeInstanceKey)
			r.staleNodes.Store(nodeID, nil)
			return true
		})
	}
}
