package archimedes

import (
	"sync"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
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
		instanceId := key.(typeInstancesMapKey)
		instance := value.(typeInstancesMapValue)

		instanceCopy := *instance
		instanceCopy.Local = false
		instances[instanceId] = &instanceCopy

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
		instanceId := key.(typeInstancesMapKey)
		instance := value.(typeInstancesMapValue)
		instances[instanceId] = instance

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

func (st *deploymentsTable) updateDeployment(deploymentId string, newEntry *api.DeploymentsTableEntryDTO) bool {
	value, ok := st.deploymentsMap.Load(deploymentId)
	if !ok {
		log.Panicf("deployment %s doesnt exist", deploymentId)
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
		instanceId := key.(typeInstancesMapKey)
		_, ok = newEntry.Instances[instanceId]
		if !ok {
			st.instancesMap.Delete(instanceId)
		}

		return true
	})

	newInstancesMap := &sync.Map{}
	for instanceId, instance := range newEntry.Instances {
		newInstancesMap.Store(instanceId, instance)
		st.instancesMap.Store(instanceId, instance)
	}

	entry.Instances = newInstancesMap
	entry.NumberOfHops = newEntry.NumberOfHops
	entry.Version = newEntry.Version
	entry.MaxHops = maxHops

	log.Debugf("updated deployment %s table entry to: %+v", deploymentId, entry)
	log.Debugf("with instances %+v", newEntry.Instances)

	return true
}

func (st *deploymentsTable) addDeployment(deploymentId string, newEntry *api.DeploymentsTableEntryDTO) (added bool) {
	_, ok := st.deploymentsMap.Load(deploymentId)
	if ok {
		added = false
		return
	}

	st.addLock.Lock()
	_, ok = st.deploymentsMap.Load(deploymentId)
	if ok {
		st.addLock.Unlock()
		added = false
		return
	}

	newTableEntry := newTempDeploymentTableEntry()
	newTableEntry.EntryLock.Lock()
	defer newTableEntry.EntryLock.Unlock()
	st.deploymentsMap.Store(deploymentId, newTableEntry)
	st.addLock.Unlock()

	newTableEntry.Host = newEntry.Host
	newTableEntry.Deployment = newEntry.Deployment

	newInstancesMap := &sync.Map{}
	for instanceId, instance := range newEntry.Instances {
		newInstancesMap.Store(instanceId, instance)
		st.instancesMap.Store(instanceId, instance)
	}

	newTableEntry.Instances = newInstancesMap
	newTableEntry.NumberOfHops = newEntry.NumberOfHops
	newTableEntry.Version = newEntry.Version
	newTableEntry.MaxHops = maxHops

	deploymentsMap := &sync.Map{}
	deploymentsMap.Store(deploymentId, struct{}{})
	st.neighborsDeploymentsMap.Store(newEntry.Host, deploymentsMap)

	added = true

	log.Debugf("added new table entry for deployment %s: %+v", deploymentId, newTableEntry)
	log.Debugf("with instances %+v", newEntry.Instances)

	return
}

func (st *deploymentsTable) getDeployment(deploymentId string) (deployment *api.Deployment, ok bool) {
	value, ok := st.deploymentsMap.Load(deploymentId)
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
		deploymentId := key.(string)
		entry := value.(typeDeploymentsTableMapValue)
		entry.EntryLock.RLock()
		defer entry.EntryLock.RUnlock()

		deployments[deploymentId] = entry.Deployment

		return true
	})

	return deployments
}

func (st *deploymentsTable) getAllDeploymentInstances(deploymentId string) map[string]*api.Instance {
	instances := map[string]*api.Instance{}

	value, ok := st.deploymentsMap.Load(deploymentId)
	if !ok {
		return instances
	}

	entry := value.(typeDeploymentsTableMapValue)
	entry.EntryLock.RLock()
	defer entry.EntryLock.RUnlock()

	entry.Instances.Range(func(key, value interface{}) bool {
		instanceId := key.(string)
		instance := value.(typeInstancesMapValue)

		instances[instanceId] = instance

		return true
	})

	return instances
}

func (st *deploymentsTable) addInstance(deploymentId, instanceId string, instance *api.Instance) (added bool) {
	value, ok := st.deploymentsMap.Load(deploymentId)
	if !ok {
		added = false
		return
	}

	entry := value.(typeDeploymentsTableMapValue)
	entry.EntryLock.Lock()
	defer entry.EntryLock.Unlock()

	entry.Instances.Store(instanceId, instance)
	entry.Version++

	st.instancesMap.Store(instanceId, instance)

	added = true
	return
}

func (st *deploymentsTable) deploymentHasInstance(deploymentId, instanceId string) bool {
	value, ok := st.deploymentsMap.Load(deploymentId)
	if !ok {
		return false
	}

	entry := value.(typeDeploymentsTableMapValue)

	entry.EntryLock.RLock()
	defer entry.EntryLock.RUnlock()

	_, ok = entry.Instances.Load(instanceId)

	return ok
}

func (st *deploymentsTable) getDeploymentInstance(deploymentId, instanceId string) (*api.Instance, bool) {
	value, ok := st.deploymentsMap.Load(deploymentId)
	if !ok {
		return nil, false
	}

	entry := value.(typeDeploymentsTableMapValue)

	entry.EntryLock.RLock()
	defer entry.EntryLock.RUnlock()

	value, ok = entry.Instances.Load(instanceId)
	if !ok {
		return nil, false
	}

	return value.(typeInstancesMapValue), ok
}

func (st *deploymentsTable) getInstance(instanceId string) (instance *api.Instance, ok bool) {
	value, ok := st.instancesMap.Load(instanceId)
	if !ok {
		return nil, false
	}

	return value.(typeInstancesMapValue), true
}

func (st *deploymentsTable) deleteDeployment(deploymentId string) {
	value, ok := st.deploymentsMap.Load(deploymentId)
	if !ok {
		return
	}

	entry := value.(typeDeploymentsTableMapValue)
	entry.EntryLock.RLock()
	defer entry.EntryLock.RUnlock()

	entry.Instances.Range(func(key, _ interface{}) bool {
		instanceId := key.(typeInstancesMapKey)
		st.deleteInstance(deploymentId, instanceId)
		return true
	})

	st.deploymentsMap.Delete(deploymentId)
}

func (st *deploymentsTable) deleteInstance(deploymentId, instanceId string) {
	value, ok := st.instancesMap.Load(deploymentId)
	if ok {
		entry := value.(typeDeploymentsTableMapValue)
		entry.EntryLock.RLock()
		entry.Instances.Delete(instanceId)
		numInstances := 0
		entry.Instances.Range(func(key, value interface{}) bool {
			numInstances++
			return true
		})

		if numInstances == 0 {
			log.Debugf("no instances left, deleting deployment %s", deploymentId)
			defer st.deleteDeployment(deploymentId)
		}

		entry.EntryLock.RUnlock()
	}

	st.instancesMap.Delete(instanceId)
}

func (st *deploymentsTable) toChangedDiscoverMsg() *api.DiscoverMsg {
	entries := map[string]*api.DeploymentsTableEntryDTO{}

	st.deploymentsMap.Range(func(key, value interface{}) bool {
		deploymentId := key.(typeDeploymentsTableMapKey)
		entry := value.(typeDeploymentsTableMapValue)

		entry.EntryLock.RLock()

		if entry.NumberOfHops+1 > maxHops {
			return true
		}

		defer entry.EntryLock.RUnlock()

		entryDTO := entry.toChangedDTO()
		entryDTO.NumberOfHops++

		entries[deploymentId] = entryDTO

		return true
	})

	if len(entries) == 0 {
		return nil
	}

	return &api.DiscoverMsg{
		MessageId:    uuid.New(),
		Origin:       archimedesId,
		NeighborSent: archimedesId,
		Entries:      entries,
	}
}

func (st *deploymentsTable) toDiscoverMsg() *api.DiscoverMsg {
	entries := map[string]*api.DeploymentsTableEntryDTO{}

	st.deploymentsMap.Range(func(key, value interface{}) bool {
		deploymentId := key.(typeDeploymentsTableMapKey)
		entry := value.(typeDeploymentsTableMapValue)

		entryDTO := entry.toDTO()
		entries[deploymentId] = entryDTO

		return true
	})

	if len(entries) == 0 {
		return nil
	}

	return &api.DiscoverMsg{
		MessageId:    uuid.New(),
		Origin:       archimedesId,
		NeighborSent: archimedesId,
		Entries:      entries,
	}
}

func (st *deploymentsTable) deleteNeighborDeployments(neighborId string) {
	value, ok := st.neighborsDeploymentsMap.Load(neighborId)
	if !ok {
		return
	}

	deployments := value.(typeNeighborsDeploymentsMapValue)
	deployments.Range(func(key, _ interface{}) bool {
		deploymentId := key.(typeNeighborsDeploymentsMapKey)
		sTable.deleteDeployment(deploymentId)
		return true
	})
}
