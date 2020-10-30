package clients

import (
	"sync"
	"time"

	archimedesHTTPClient "github.com/bruno-anjos/archimedesHTTPClient"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/archimedes/cell"
	"github.com/golang/geo/s2"
)

type (
	batchValue struct {
		Locations map[s2.CellID]int
		NumReqs   int
	}

	Manager struct {
		numReqsLastMinute map[string]*batchValue
		currBatch         map[string]*batchValue
		numReqsLock       sync.RWMutex
		addDeploymentLock sync.Mutex
		exploringCells    sync.Map
		cellManager       *cell.Manager
	}
)

const (
	batchTimer = 10 * time.Second
)

func NewManager() *Manager {
	r := &Manager{
		numReqsLastMinute: map[string]*batchValue{},
		currBatch:         map[string]*batchValue{},
		numReqsLock:       sync.RWMutex{},
		addDeploymentLock: sync.Mutex{},
		exploringCells:    sync.Map{},
		cellManager:       cell.NewManager(),
	}

	go r.manageLoadBatch()

	return r
}

func (r *Manager) UpdateNumRequests(deploymentId string, location s2.CellID) {
	r.numReqsLock.Lock()
	defer r.numReqsLock.Unlock()

	entry, ok := r.numReqsLastMinute[deploymentId]
	if !ok {
		r.addNewEntry(deploymentId, location)
	} else {
		r.updateEntry(entry, location)
	}

	entry, ok = r.currBatch[deploymentId]
	if !ok {
		r.addNewBatch(deploymentId, location)
	} else {
		r.updateBatch(entry, location)
	}

	r.cellManager.AddClientToDownmostCell(deploymentId, location)
}

func (r *Manager) addNewEntry(deploymentId string, locId s2.CellID) {
	r.numReqsLastMinute[deploymentId] = &batchValue{
		Locations: map[s2.CellID]int{
			locId: 1,
		},
		NumReqs: 1,
	}
}

func (r *Manager) updateEntry(entry *batchValue, location s2.CellID) {
	entry.NumReqs++
	entry.Locations[location]++
}

func (r *Manager) addNewBatch(deploymentId string, location s2.CellID) {
	r.currBatch[deploymentId] = &batchValue{
		Locations: map[s2.CellID]int{
			location: 1,
		},
		NumReqs: 1,
	}
}

func (r *Manager) updateBatch(entry *batchValue, location s2.CellID) {
	entry.NumReqs++
	entry.Locations[location]++
}

func (r *Manager) GetLoad(deploymentId string) (load int) {
	r.numReqsLock.RLock()
	entry, ok := r.numReqsLastMinute[deploymentId]
	if ok {
		load = entry.NumReqs
	}
	r.numReqsLock.RUnlock()

	return
}

func (r *Manager) AddToExploring(deploymentId string, cell s2.CellID) {
	r.exploringCells.Store(deploymentId, cell)
}

func (r *Manager) RemoveFromExploring(deploymentId string) {
	r.exploringCells.Delete(deploymentId)
}

func (r *Manager) GetDeploymentClientsCentroids(deploymentId string) ([]s2.CellID, bool) {
	return r.cellManager.GetDeploymentCentroids(deploymentId)
}

func (r *Manager) manageLoadBatch() {
	ticker := time.NewTicker(batchTimer)

	for {
		<-ticker.C
		r.numReqsLock.Lock()
		for deploymentId, depBatch := range r.currBatch {
			go r.waitToRemove(deploymentId, depBatch)
			r.currBatch[deploymentId] = &batchValue{
				Locations: map[s2.CellID]int{},
				NumReqs:   0,
			}
		}
		r.numReqsLock.Unlock()
	}
}

func (r *Manager) waitToRemove(deploymentId string, entry *batchValue) {
	time.Sleep(archimedesHTTPClient.CacheExpiringTime)
	r.numReqsLock.Lock()
	r.numReqsLastMinute[deploymentId].NumReqs -= entry.NumReqs
	for locId, amount := range entry.Locations {
		r.numReqsLastMinute[deploymentId].Locations[locId] -= amount
		if r.numReqsLastMinute[deploymentId].Locations[locId] == 0 {
			delete(r.numReqsLastMinute[deploymentId].Locations, locId)
		}
	}
	r.numReqsLock.Unlock()

	r.cellManager.RemoveClientsFromCells(deploymentId, entry.Locations)
}
