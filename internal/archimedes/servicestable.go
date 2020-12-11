package archimedes

import (
	"sync"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type (
	deploymentsTableEntry struct {
		Host         *utils.Node
		Deployment   *api.Deployment
		Instances    *sync.Map
		NumberOfHops int
		MaxHops      int
		Version      int
		EntryLock    *sync.RWMutex
	}
)

func newTempDeploymentTableEntry() *deploymentsTableEntry {
	return &deploymentsTableEntry{
		Host:         nil,
		Deployment:   nil,
		Instances:    nil,
		NumberOfHops: 0,
		MaxHops:      0,
		Version:      0,
		EntryLock:    &sync.RWMutex{},
	}
}

func (se *deploymentsTableEntry) toChangedDTO() *api.DeploymentsTableEntryDTO {
	instances := map[string]*api.Instance{}

	se.EntryLock.RLock()
	defer se.EntryLock.RUnlock()

	se.Instances.Range(func(key, value interface{}) bool {
		instanceID := key.(typeInstancesMapKey)
		instance := value.(typeInstancesMapValue)

		instanceCopy := *instance
		instanceCopy.Local = false
		instances[instanceID] = &instanceCopy

		return true
	})

	return &api.DeploymentsTableEntryDTO{
		Host:         se.Host,
		Deployment:   se.Deployment,
		Instances:    instances,
		NumberOfHops: se.NumberOfHops,
		MaxHops:      se.MaxHops,
		Version:      se.Version,
	}
}

func (se *deploymentsTableEntry) toDTO() *api.DeploymentsTableEntryDTO {
	instances := map[string]*api.Instance{}

	se.EntryLock.RLock()
	defer se.EntryLock.RUnlock()

	se.Instances.Range(func(key, value interface{}) bool {
		instanceID := key.(typeInstancesMapKey)
		instance := value.(typeInstancesMapValue)
		instances[instanceID] = instance

		return true
	})

	return &api.DeploymentsTableEntryDTO{
		Host:         se.Host,
		Deployment:   se.Deployment,
		Instances:    instances,
		NumberOfHops: se.NumberOfHops,
		MaxHops:      se.MaxHops,
		Version:      se.Version,
	}
}

type (
	deploymentsTable struct {
		addLock                 sync.Mutex
		deploymentsMap          sync.Map
		instancesMap            sync.Map
		neighborsDeploymentsMap sync.Map
	}

	typeDeploymentsTableMapKey   = string
	typeDeploymentsTableMapValue = *deploymentsTableEntry

	typeInstancesMapKey   = string
	typeInstancesMapValue = *api.Instance

	typeNeighborsDeploymentsMapKey   = string
	typeNeighborsDeploymentsMapValue = *sync.Map
)

func newDeploymentsTable() *deploymentsTable {
	return &deploymentsTable{
		addLock:                 sync.Mutex{},
		deploymentsMap:          sync.Map{},
		instancesMap:            sync.Map{},
		neighborsDeploymentsMap: sync.Map{},
	}
}

func (st *deploymentsTable) updateDeployment(deploymentID string, newEntry *api.DeploymentsTableEntryDTO) bool {
	value, ok := st.deploymentsMap.Load(deploymentID)
	if !ok {
		log.Panicf("deployment %s doesnt exist", deploymentID)
	}

	entry := value.(typeDeploymentsTableMapValue)
	entry.EntryLock.RLock()

	log.Debugf("got deployment on version %d, have %d", entry.Version, newEntry.Version)

	// ignore messages with no new information
	if newEntry.Version <= entry.Version {
		log.Debug("discarding message due to version being older or equal")

		return false
	}

	entry.EntryLock.RUnlock()
	entry.EntryLock.Lock()
	defer entry.EntryLock.Unlock()

	// message is fresher, comes from the closest neighbor or closer and it has new information
	entry.Host = newEntry.Host
	entry.Deployment = newEntry.Deployment

	entry.Instances.Range(func(key, value interface{}) bool {
		instanceID := key.(typeInstancesMapKey)
		_, ok = newEntry.Instances[instanceID]
		if !ok {
			st.instancesMap.Delete(instanceID)
		}

		return true
	})

	newInstancesMap := &sync.Map{}

	for instanceID, instance := range newEntry.Instances {
		newInstancesMap.Store(instanceID, instance)
		st.instancesMap.Store(instanceID, instance)
	}

	entry.Instances = newInstancesMap
	entry.NumberOfHops = newEntry.NumberOfHops
	entry.Version = newEntry.Version
	entry.MaxHops = maxHops

	log.Debugf("updated deployment %s table entry to: %+v", deploymentID, entry)
	log.Debugf("with instances %+v", newEntry.Instances)

	return true
}

func (st *deploymentsTable) addDeployment(deploymentID string, newEntry *api.DeploymentsTableEntryDTO) (added bool) {
	_, ok := st.deploymentsMap.Load(deploymentID)
	if ok {
		added = false

		return
	}

	st.addLock.Lock()

	_, ok = st.deploymentsMap.Load(deploymentID)
	if ok {
		st.addLock.Unlock()

		added = false

		return
	}

	newTableEntry := newTempDeploymentTableEntry()
	newTableEntry.EntryLock.Lock()
	defer newTableEntry.EntryLock.Unlock()
	st.deploymentsMap.Store(deploymentID, newTableEntry)
	st.addLock.Unlock()

	newTableEntry.Host = newEntry.Host
	newTableEntry.Deployment = newEntry.Deployment

	newInstancesMap := &sync.Map{}

	for instanceID, instance := range newEntry.Instances {
		newInstancesMap.Store(instanceID, instance)
		st.instancesMap.Store(instanceID, instance)
	}

	newTableEntry.Instances = newInstancesMap
	newTableEntry.NumberOfHops = newEntry.NumberOfHops
	newTableEntry.Version = newEntry.Version
	newTableEntry.MaxHops = maxHops

	deploymentsMap := &sync.Map{}
	deploymentsMap.Store(deploymentID, struct{}{})
	st.neighborsDeploymentsMap.Store(newEntry.Host, deploymentsMap)

	added = true

	log.Debugf("added new table entry for deployment %s: %+v", deploymentID, newTableEntry)
	log.Debugf("with instances %+v", newEntry.Instances)

	return added
}

func (st *deploymentsTable) getDeployment(deploymentID string) (deployment *api.Deployment, ok bool) {
	value, ok := st.deploymentsMap.Load(deploymentID)
	if !ok {
		return nil, false
	}

	entry := value.(typeDeploymentsTableMapValue)
	entry.EntryLock.RLock()
	defer entry.EntryLock.RUnlock()

	return entry.Deployment, true
}

func (st *deploymentsTable) getAllDeployments() map[string]*api.Deployment {
	deployments := map[string]*api.Deployment{}

	st.deploymentsMap.Range(func(key, value interface{}) bool {
		deploymentID := key.(string)
		entry := value.(typeDeploymentsTableMapValue)
		entry.EntryLock.RLock()
		defer entry.EntryLock.RUnlock()

		deployments[deploymentID] = entry.Deployment

		return true
	})

	return deployments
}

func (st *deploymentsTable) getAllDeploymentInstances(deploymentID string) map[string]*api.Instance {
	instances := map[string]*api.Instance{}

	value, ok := st.deploymentsMap.Load(deploymentID)
	if !ok {
		return instances
	}

	entry := value.(typeDeploymentsTableMapValue)
	entry.EntryLock.RLock()
	defer entry.EntryLock.RUnlock()

	entry.Instances.Range(func(key, value interface{}) bool {
		instanceID := key.(string)
		instance := value.(typeInstancesMapValue)

		instances[instanceID] = instance

		return true
	})

	return instances
}

func (st *deploymentsTable) addInstance(deploymentID, instanceID string, instance *api.Instance) (added bool) {
	value, ok := st.deploymentsMap.Load(deploymentID)
	if !ok {
		added = false

		return
	}

	entry := value.(typeDeploymentsTableMapValue)
	entry.EntryLock.Lock()
	defer entry.EntryLock.Unlock()

	entry.Instances.Store(instanceID, instance)
	entry.Version++

	st.instancesMap.Store(instanceID, instance)

	added = true

	return added
}

func (st *deploymentsTable) deploymentHasInstance(deploymentID, instanceID string) bool {
	value, ok := st.deploymentsMap.Load(deploymentID)
	if !ok {
		return false
	}

	entry := value.(typeDeploymentsTableMapValue)

	entry.EntryLock.RLock()
	defer entry.EntryLock.RUnlock()

	_, ok = entry.Instances.Load(instanceID)

	return ok
}

func (st *deploymentsTable) getDeploymentInstance(deploymentID, instanceID string) (*api.Instance, bool) {
	value, ok := st.deploymentsMap.Load(deploymentID)
	if !ok {
		return nil, false
	}

	entry := value.(typeDeploymentsTableMapValue)

	entry.EntryLock.RLock()
	defer entry.EntryLock.RUnlock()

	value, ok = entry.Instances.Load(instanceID)
	if !ok {
		return nil, false
	}

	return value.(typeInstancesMapValue), ok
}

func (st *deploymentsTable) getInstance(instanceID string) (instance *api.Instance, ok bool) {
	value, ok := st.instancesMap.Load(instanceID)
	if !ok {
		return nil, false
	}

	return value.(typeInstancesMapValue), true
}

func (st *deploymentsTable) deleteDeployment(deploymentID string) {
	value, ok := st.deploymentsMap.Load(deploymentID)
	if !ok {
		return
	}

	entry := value.(typeDeploymentsTableMapValue)
	entry.EntryLock.RLock()
	defer entry.EntryLock.RUnlock()

	entry.Instances.Range(func(key, _ interface{}) bool {
		instanceID := key.(typeInstancesMapKey)
		st.deleteInstance(deploymentID, instanceID)

		return true
	})

	st.deploymentsMap.Delete(deploymentID)
}

func (st *deploymentsTable) deleteInstance(deploymentID, instanceID string) {
	value, ok := st.instancesMap.Load(deploymentID)
	if ok {
		entry := value.(typeDeploymentsTableMapValue)
		entry.EntryLock.RLock()
		entry.Instances.Delete(instanceID)

		numInstances := 0

		entry.Instances.Range(func(key, value interface{}) bool {
			numInstances++

			return true
		})

		if numInstances == 0 {
			log.Debugf("no instances left, deleting deployment %s", deploymentID)
			defer st.deleteDeployment(deploymentID)
		}

		entry.EntryLock.RUnlock()
	}

	st.instancesMap.Delete(instanceID)
}

func (st *deploymentsTable) toChangedDiscoverMsg() *api.DiscoverMsg {
	entries := map[string]*api.DeploymentsTableEntryDTO{}

	st.deploymentsMap.Range(func(key, value interface{}) bool {
		deploymentID := key.(typeDeploymentsTableMapKey)
		entry := value.(typeDeploymentsTableMapValue)

		entry.EntryLock.RLock()

		if entry.NumberOfHops+1 > maxHops {
			return true
		}

		defer entry.EntryLock.RUnlock()

		entryDTO := entry.toChangedDTO()
		entryDTO.NumberOfHops++

		entries[deploymentID] = entryDTO

		return true
	})

	if len(entries) == 0 {
		return nil
	}

	return &api.DiscoverMsg{
		MessageID:    uuid.New(),
		Origin:       archimedesID,
		NeighborSent: archimedesID,
		Entries:      entries,
	}
}

func (st *deploymentsTable) toDiscoverMsg() *api.DiscoverMsg {
	entries := map[string]*api.DeploymentsTableEntryDTO{}

	st.deploymentsMap.Range(func(key, value interface{}) bool {
		deploymentID := key.(typeDeploymentsTableMapKey)
		entry := value.(typeDeploymentsTableMapValue)

		entryDTO := entry.toDTO()
		entries[deploymentID] = entryDTO

		return true
	})

	if len(entries) == 0 {
		return nil
	}

	return &api.DiscoverMsg{
		MessageID:    uuid.New(),
		Origin:       archimedesID,
		NeighborSent: archimedesID,
		Entries:      entries,
	}
}

func (st *deploymentsTable) deleteNeighborDeployments(neighborID string) {
	value, ok := st.neighborsDeploymentsMap.Load(neighborID)
	if !ok {
		return
	}

	deployments := value.(typeNeighborsDeploymentsMapValue)
	deployments.Range(func(key, _ interface{}) bool {
		deploymentID := key.(typeNeighborsDeploymentsMapKey)
		sTable.deleteDeployment(deploymentID)

		return true
	})
}
