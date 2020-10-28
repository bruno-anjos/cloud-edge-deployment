package clients

import (
	"sync"
	"time"

	archimedesHTTPClient "github.com/bruno-anjos/archimedesHTTPClient"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/archimedes/cell"
	publicUtils "github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/golang/geo/s2"
)

type (
	batchValue struct {
		Locations map[string]*cell.LocationsEntry
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

func (r *Manager) UpdateNumRequests(deploymentId string, location *publicUtils.Location) {
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

func (r *Manager) addNewEntry(deploymentId string, location *publicUtils.Location) {
	r.numReqsLastMinute[deploymentId] = &batchValue{
		Locations: map[string]*cell.LocationsEntry{
			location.GetId(): {
				Location: location,
				Number:   1,
			},
		},
		NumReqs: 1,
	}
}

func (r *Manager) updateEntry(entry *batchValue, location *publicUtils.Location) {
	entry.NumReqs++

	var (
		loc *cell.LocationsEntry
	)
	loc, ok := entry.Locations[location.GetId()]
	if !ok {
		entry.Locations[location.GetId()] = &cell.LocationsEntry{
			Location: location,
			Number:   1,
		}
	} else {
		loc.Number++
	}
}

func (r *Manager) addNewBatch(deploymentId string, location *publicUtils.Location) {
	r.currBatch[deploymentId] = &batchValue{
		Locations: map[string]*cell.LocationsEntry{
			location.GetId(): {
				Location: location,
				Number:   1,
			},
		},
		NumReqs: 1,
	}
}

func (r *Manager) updateBatch(entry *batchValue, location *publicUtils.Location) {
	entry.NumReqs++

	var (
		loc *cell.LocationsEntry
	)
	loc, ok := entry.Locations[location.GetId()]
	if !ok {
		entry.Locations[location.GetId()] = &cell.LocationsEntry{
			Location: location,
			Number:   1,
		}
	} else {
		loc.Number++
	}
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

func (r *Manager) AddToExploring(deploymentId string, cells []s2.CellID) {
	r.exploringCells.Store(deploymentId, cells)
}

func (r *Manager) RemoveFromExploring(deploymentId string) {
	r.exploringCells.Delete(deploymentId)
}

func (r *Manager) GetDeploymentClientsCentroids(deploymentId string) ([]*publicUtils.Location, bool) {
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
				Locations: map[string]*cell.LocationsEntry{},
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
	for locId, locEntry := range entry.Locations {
		r.numReqsLastMinute[deploymentId].Locations[locId].Number -= locEntry.Number
		if r.numReqsLastMinute[deploymentId].Locations[locId].Number == 0 {
			delete(r.numReqsLastMinute[deploymentId].Locations, locId)
		}
	}
	r.numReqsLock.Unlock()

	r.cellManager.RemoveClientsFromCells(deploymentId, entry.Locations)
}
