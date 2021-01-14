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
		Deployment *api.Deployment
		Instances  *sync.Map
		MaxHops    int
		EntryLock  *sync.RWMutex
	}
)

func newTempDeploymentTableEntry() *deploymentsTableEntry {
	return &deploymentsTableEntry{
		Deployment: nil,
		Instances:  nil,
		MaxHops:    0,
		EntryLock:  &sync.RWMutex{},
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
		Deployment: se.Deployment,
		Instances:  instances,
		MaxHops:    se.MaxHops,
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
		Deployment: se.Deployment,
		Instances:  instances,
		MaxHops:    se.MaxHops,
	}
}

type (
	deploymentsTable struct {
		addLock        sync.Mutex
		deploymentsMap sync.Map
		instancesMap   sync.Map
	}

	typeDeploymentsTableMapKey   = string
	typeDeploymentsTableMapValue = *deploymentsTableEntry

	typeInstancesMapKey   = string
	typeInstancesMapValue = *api.Instance
)

func newDeploymentsTable() *deploymentsTable {
	return &deploymentsTable{
		addLock:        sync.Mutex{},
		deploymentsMap: sync.Map{},
		instancesMap:   sync.Map{},
	}
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

	newTableEntry.Deployment = newEntry.Deployment

	newInstancesMap := &sync.Map{}

	for instanceID, instance := range newEntry.Instances {
		newInstancesMap.Store(instanceID, instance)
		st.instancesMap.Store(instanceID, instance)
	}

	newTableEntry.Instances = newInstancesMap
	newTableEntry.MaxHops = maxHops

	deploymentsMap := &sync.Map{}
	deploymentsMap.Store(deploymentID, struct{}{})

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

func (st *deploymentsTable) getAllLocalDeploymentInstances(deploymentID string) map[string]*api.Instance {
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

		if instance.Host.ID == myself.ID {
			instances[instanceID] = instance
		}

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
		st.deleteInstance(instanceID)

		return true
	})

	st.deploymentsMap.Delete(deploymentID)
}

func (st *deploymentsTable) deleteDeploymentInstancesFrom(deploymentID string, from *utils.Node) {
	value, ok := st.deploymentsMap.Load(deploymentID)
	if !ok {
		return
	}

	entry := value.(typeDeploymentsTableMapValue)
	entry.EntryLock.Lock()
	defer entry.EntryLock.Unlock()

	entry.Instances.Range(func(key, value interface{}) bool {
		instanceID := key.(typeInstancesMapKey)
		instance := value.(typeInstancesMapValue)
		if instance.ID == from.ID {
			entry.Instances.Delete(instanceID)
		}

		return true
	})
}

func (st *deploymentsTable) deleteInstance(instanceID string) {
	value, ok := st.instancesMap.Load(instanceID)
	if ok {
		instance := value.(typeInstancesMapValue)
		value, ok = st.deploymentsMap.Load(instance.DeploymentID)
		if ok {
			entry := value.(typeDeploymentsTableMapValue)
			entry.EntryLock.RLock()

			numInstances := 0

			entry.Instances.Range(func(key, value interface{}) bool {
				numInstances++

				return true
			})

			if numInstances == 0 {
				log.Debugf("no instances left, deleting deployment %s", instance.DeploymentID)
				defer st.deleteDeployment(instance.DeploymentID)
			}

			entry.EntryLock.RUnlock()
		}
	}

	st.instancesMap.Delete(instanceID)
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
		MessageID: uuid.New(),
		Origin:    myself,
		Entries:   entries,
	}
}

func (st *deploymentsTable) toChangedDiscoverMsg() *api.DiscoverMsg {
	entries := map[string]*api.DeploymentsTableEntryDTO{}

	st.deploymentsMap.Range(func(key, value interface{}) bool {
		deploymentID := key.(typeDeploymentsTableMapKey)
		entry := value.(typeDeploymentsTableMapValue)

		entryDTO := entry.toChangedDTO()
		entries[deploymentID] = entryDTO

		return true
	})

	if len(entries) == 0 {
		return nil
	}

	return &api.DiscoverMsg{
		MessageID: uuid.New(),
		Origin:    myself,
		Entries:   entries,
	}
}
