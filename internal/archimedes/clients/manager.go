package clients

import (
	"sync"
	"sync/atomic"
	"time"

	archimedesHTTPClient "github.com/bruno-anjos/archimedesHTTPClient"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/archimedes/cell"
	"github.com/golang/geo/s2"
)

type (
	batchValue struct {
		Locations *sync.Map
		NumReqs   int64
	}

	exploringCellsValueType = []s2.CellID

	currBatchMapKey   = string
	currBatchMapValue = *batchValue

	numReqsLastMinuteMapValue = *batchValue

	Manager struct {
		numReqsLastMinute sync.Map
		currBatch         sync.Map
		exploringCells    sync.Map
		cellManager       *cell.Manager
	}
)

const (
	batchTimer = 10 * time.Second
)

func NewManager() *Manager {
	r := &Manager{
		numReqsLastMinute: sync.Map{},
		currBatch:         sync.Map{},
		exploringCells:    sync.Map{},
		cellManager:       cell.NewManager(),
	}

	go r.manageLoadBatch()

	return r
}

func (r *Manager) AddDeployment(deploymentId string) {
	reqsLastMinute := &batchValue{
		Locations: &sync.Map{},
		NumReqs:   0,
	}
	r.numReqsLastMinute.LoadOrStore(deploymentId, reqsLastMinute)

	newBatch := &batchValue{
		Locations: &sync.Map{},
		NumReqs:   0,
	}
	r.currBatch.LoadOrStore(deploymentId, newBatch)
}

func (r *Manager) UpdateNumRequests(deploymentId string, location s2.CellID) {
	r.updateEntry(deploymentId, location)
	r.updateBatch(deploymentId, location)

	r.cellManager.AddClientToDownmostCell(deploymentId, location)
}

// Even though this is thread safe, this does not guarantee a perfectly accurate count
// of requests received since one can load the entry, meanwhile the entry be swapped, and
// increment the entry that is stale, thus never reflecting the count on the updated entry.
func (r *Manager) updateEntry(deploymentId string, location s2.CellID) {
	// load the entry
	value, ok := r.numReqsLastMinute.Load(deploymentId)
	if !ok {
		return
	}

	// possibly increment an entry that is already stale
	entry := value.(numReqsLastMinuteMapValue)

	var intValue = new(int64)
	value, _ = entry.Locations.LoadOrStore(location, intValue)
	intValue = value.(cell.LocationsMapValue)

	atomic.AddInt64(intValue, 1)
	atomic.AddInt64(&entry.NumReqs, 1)
}

// Same as updateEntry
func (r *Manager) updateBatch(deploymentId string, location s2.CellID) {
	value, ok := r.numReqsLastMinute.Load(deploymentId)
	if !ok {
		return
	}

	entry := value.(currBatchMapValue)

	var intValue = new(int64)
	value, _ = entry.Locations.LoadOrStore(location, intValue)
	intValue = value.(cell.LocationsMapValue)

	atomic.AddInt64(intValue, 1)
	atomic.AddInt64(&entry.NumReqs, 1)
}

func (r *Manager) GetLoad(deploymentId string) (load int) {
	value, ok := r.numReqsLastMinute.Load(deploymentId)
	if !ok {
		return 0
	}

	entry := value.(numReqsLastMinuteMapValue)
	if ok {
		load = int(entry.NumReqs)
	}

	return
}

func (r *Manager) SetToExploring(deploymentId string, cells []s2.CellID) {
	r.exploringCells.Store(deploymentId, cells)
}

func (r *Manager) RemoveFromExploring(deploymentId string) {
	r.exploringCells.Delete(deploymentId)
}

func (r *Manager) GetDeploymentClientsCentroids(deploymentId string) (cells []s2.CellID, ok bool) {
	cells, ok = r.cellManager.GetDeploymentCentroids(deploymentId)
	if len(cells) == 0 || !ok {
		value, eOk := r.exploringCells.Load(deploymentId)
		if eOk {
			cells = value.(exploringCellsValueType)
		}
		ok = eOk
	}

	return
}

func (r *Manager) manageLoadBatch() {
	ticker := time.NewTicker(batchTimer)

	for {
		<-ticker.C

		r.currBatch.Range(func(key, value interface{}) bool {
			deploymentId := key.(currBatchMapKey)
			depBatch := value.(currBatchMapValue)

			newBatch := &batchValue{
				Locations: &sync.Map{},
				NumReqs:   0,
			}
			r.currBatch.Store(deploymentId, newBatch)
			go r.waitToRemove(deploymentId, depBatch)

			return true
		})
	}
}

func (r *Manager) waitToRemove(deploymentId string, batch *batchValue) {
	time.Sleep(archimedesHTTPClient.CacheExpiringTime)

	// load numRequests and decrement the amount of requests by the amount of requests in this batch
	value, ok := r.numReqsLastMinute.Load(deploymentId)
	if !ok {
		return
	}
	entry := value.(numReqsLastMinuteMapValue)
	atomic.AddInt64(&entry.NumReqs, -batch.NumReqs)

	// iterate this batch locations and decrement the count of each location in numRequests by the amount
	// of each location on this batch
	entry.Locations.Range(func(key, value interface{}) bool {
		locId := key.(cell.LocationsMapKey)
		amount := value.(cell.LocationsMapValue)
		value, ok = r.numReqsLastMinute.Load(deploymentId)
		if !ok {
			return false
		}

		entry = value.(numReqsLastMinuteMapValue)
		value, ok = entry.Locations.Load(locId)
		if !ok {
			return false
		}

		intValue := value.(cell.LocationsMapValue)

		atomic.AddInt64(intValue, -atomic.LoadInt64(amount))
		return true
	})

	r.cellManager.RemoveClientsFromCells(deploymentId, entry.Locations)
}
