package archimedes

import (
	"sync"
	"time"

	archimedesHTTPClient "github.com/bruno-anjos/archimedesHTTPClient"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/archimedes/cell_manager"
	publicUtils "github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/golang/geo/s2"
)

type (
	cellsByDeploymentValue = *cell_manager.CellsCollection

	exploringCellsValue = []s2.CellID

	activeCell struct {
		cells map[s2.CellID]interface{}
		lock  sync.Mutex
	}
	activeCellsValue = *activeCell

	reqsLocationManager struct {
		numReqsLastMinute map[string]*batchValue
		currBatch         map[string]*batchValue
		numReqsLock       sync.RWMutex
		cellsByDeployment sync.Map
		addDeploymentLock sync.Mutex
		exploringCells    sync.Map
		activeCells       sync.Map
	}
)

const (
	minCellLevel = 8
	maxCellLevel = 16

	maxClientsInCell = 300

	timeBetweenSplitAttempts = 5 * time.Second
)

func newReqsLocationManager() *reqsLocationManager {
	r := &reqsLocationManager{
		numReqsLastMinute: map[string]*batchValue{},
		currBatch:         map[string]*batchValue{},
		numReqsLock:       sync.RWMutex{},
		cellsByDeployment: sync.Map{},
		addDeploymentLock: sync.Mutex{},
		exploringCells:    sync.Map{},
		activeCells:       sync.Map{},
	}

	go r.manageLoadBatch()

	return r
}

func (r *reqsLocationManager) updateNumRequests(deploymentId string, location *publicUtils.Location) {
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

	r.updateCellsWithClientLocation(deploymentId, location)
}

func (r *reqsLocationManager) addNewEntry(deploymentId string, location *publicUtils.Location) {
	r.numReqsLastMinute[deploymentId] = &batchValue{
		Locations: map[string]*locationsEntry{
			location.GetId(): {
				Location: location,
				Number:   1,
			},
		},
		NumReqs: 1,
	}
}

func (r *reqsLocationManager) updateEntry(entry *batchValue, location *publicUtils.Location) {
	entry.NumReqs++

	var (
		loc *locationsEntry
	)
	loc, ok := entry.Locations[location.GetId()]
	if !ok {
		entry.Locations[location.GetId()] = &locationsEntry{
			Location: location,
			Number:   1,
		}
	} else {
		loc.Number++
	}
}

func (r *reqsLocationManager) addNewBatch(deploymentId string, location *publicUtils.Location) {
	r.currBatch[deploymentId] = &batchValue{
		Locations: map[string]*locationsEntry{
			location.GetId(): {
				Location: location,
				Number:   1,
			},
		},
		NumReqs: 1,
	}
}

func (r *reqsLocationManager) updateBatch(entry *batchValue, location *publicUtils.Location) {
	entry.NumReqs++

	var (
		loc *locationsEntry
	)
	loc, ok := entry.Locations[location.GetId()]
	if !ok {
		entry.Locations[location.GetId()] = &locationsEntry{
			Location: location,
			Number:   1,
		}
	} else {
		loc.Number++
	}
}

func (r *reqsLocationManager) getLoad(deploymentId string) (load int) {
	r.numReqsLock.RLock()
	entry, ok := r.numReqsLastMinute[deploymentId]
	if ok {
		load = entry.NumReqs
	}
	r.numReqsLock.RUnlock()

	return
}

func (r *reqsLocationManager) getDeploymentCentroids(deploymentId string) (centroids []*publicUtils.Location, ok bool) {
	var value interface{}
	value, ok = r.cellsByDeployment.Load(deploymentId)
	if !ok {
		value, ok = r.exploringCells.Load(deploymentId)
		if ok {
			exploreCell := value.(exploringCellsValue)
			for _, cellId := range exploreCell {
				cellCenter := s2.LatLngFromPoint(s2.CellFromCellID(cellId).Center())
				centroids = append(centroids, publicUtils.FromLatLngToLocation(&cellCenter))
			}
			return
		}
		return
	}

	hasLocations := false
	deploymentCells := value.(cellsByDeploymentValue)

	deploymentCells.Iterate(func(id s2.CellID, cell *cell_manager.Cell) bool {
		hasLocations = true
		cellCenter := s2.LatLngFromPoint(s2.CellFromCellID(id).Center())
		centroids = append(centroids, publicUtils.FromLatLngToLocation(&cellCenter))
		return true
	})
	ok = hasLocations

	return
}

func (r *reqsLocationManager) addToExploring(deploymentId string, cells []s2.CellID) {
	r.exploringCells.Store(deploymentId, cells)
}

func (r *reqsLocationManager) removeFromExploring(deploymentId string) {
	r.exploringCells.Delete(deploymentId)
}

func (r *reqsLocationManager) updateCellsWithClientLocation(deploymentId string, location *publicUtils.Location) {
	deploymentCells := r.getDeploymentCells(deploymentId)

	leafId := s2.CellIDFromLatLng(location.ToLatLng())

	missingTopCell := true
	deploymentCells.RLock()
	deploymentCells.Iterate(func(id s2.CellID, cell *cell_manager.Cell) bool {
		if id.Contains(leafId) {
			id, cell = r.percolateToDownmostCell(id, cell, leafId)
			cell.Lock()
			cell.AddClient(location)
			cell.Unlock()
			cell.RLock()
			if id.Level() < maxCellLevel && cell.GetNumClients() == maxClientsInCell {
				go r.splitMaxedCell(deploymentId, id, cell)
			}
			cell.RUnlock()
			missingTopCell = false
			return false
		}
		return true
	})
	deploymentCells.RUnlock()

	if missingTopCell {
		deploymentCells.Lock()
		deploymentCells.Iterate(func(id s2.CellID, cell *cell_manager.Cell) bool {
			if id.Contains(leafId) {
				id, cell = r.percolateToDownmostCell(id, cell, leafId)
				cell.Lock()
				cell.AddClient(location)
				cell.Unlock()
				cell.RLock()
				if id.Level() < maxCellLevel && cell.GetNumClients() == maxClientsInCell {
					go r.splitMaxedCell(deploymentId, id, cell)
				}
				cell.RUnlock()
				missingTopCell = false
				return false
			}
			return true
		})

		if missingTopCell {
			cell := cell_manager.NewCell(0, map[string]*cell_manager.ClientLocation{})
			cellId := s2.CellIDFromLatLng(location.ToLatLng()).Parent(minCellLevel)
			deploymentCells.AddCell(cellId, cell)
			r.activeCells.Store(cellId, cell)
			cell.AddClient(location)
		}
		deploymentCells.Unlock()
	}
}

func (r *reqsLocationManager) removeClientFromCell() {

}

func (r *reqsLocationManager) percolateToDownmostCell(cellId s2.CellID, c *cell_manager.Cell,
	leafId s2.CellID) (s2.CellID, *cell_manager.Cell) {
	if !c.HasChildren() {
		return cellId, c
	}

	level := cellId.Level()
	discoverCell := c
	resultCellId := cellId
	var locks []*sync.RWMutex
	for {
		discoverCell.RLock()

		if !c.HasChildren() || level > maxCellLevel {
			break
		}

		childContains := false
		children := discoverCell.GetChildren()

		children.RLock()
		children.Iterate(func(id s2.CellID, cell *cell_manager.Cell) bool {
			if id.Contains(leafId) {
				childContains = true
				discoverCell = cell
				resultCellId = id
				level++
				return false
			}
			return true
		})

		if !childContains {
			children.RUnlock()
			children.Lock()
			children.Iterate(func(id s2.CellID, cell *cell_manager.Cell) bool {
				if id.Contains(leafId) {
					childContains = true
					discoverCell = cell
					resultCellId = id
					level++
					return false
				}
				return true
			})

			if !childContains {
				resultCellId = leafId.Parent(level + 1)

				newChildCell := cell_manager.NewCell(0, map[string]*cell_manager.ClientLocation{})
				discoverCell.AddChild(resultCellId, newChildCell)
				r.activeCells.Store(resultCellId, newChildCell)
				discoverCell = newChildCell

				children.Unlock()
				discoverCell.RUnlock()
				break
			}
		}

		locks = append(locks, &discoverCell.RWMutex, &children.RWMutex)
	}

	for _, lock := range locks {
		lock.RUnlock()
	}

	return resultCellId, discoverCell
}

func (r *reqsLocationManager) getDeploymentCells(deploymentId string) *cell_manager.CellsCollection {
	value, ok := r.cellsByDeployment.Load(deploymentId)
	if !ok {
		r.addDeploymentLock.Lock()
		value, ok = r.cellsByDeployment.Load(deploymentId)
		if !ok {
			value = &sync.Map{}
			r.cellsByDeployment.Store(deploymentId, value)
		}
		r.addDeploymentLock.Unlock()
	}

	return value.(cellsByDeploymentValue)
}

func (r *reqsLocationManager) splitMaxedCell(deploymentId string, cellId s2.CellID, cell *cell_manager.Cell) {
	toSplitIds := []s2.CellID{cellId}
	toSplitCells := []*cell_manager.Cell{cell}

	for len(toSplitIds) != 0 {
		splittingCellId := toSplitIds[0]
		splittingCell := toSplitCells[0]

		toSplitIds = toSplitIds[1:]
		toSplitCells = toSplitCells[1:]

		newCells := map[s2.CellID]*cell_manager.Cell{}
		splittingCell.Lock()
		splittingCell.IterateLocations(func(locId string, loc *cell_manager.ClientLocation) bool {
			newCellId := s2.CellIDFromLatLng(loc.Location.ToLatLng()).Parent(splittingCellId.Level() + 1)
			tempCell, ok := newCells[newCellId]
			if !ok {
				newTempCell := cell_manager.NewCell(loc.Count, map[string]*cell_manager.ClientLocation{locId: loc})
				newCells[newCellId] = newTempCell
			} else {
				tempCell.AddClient(loc.Location)
			}
			return true
		})

		for tempCellId, tempCell := range newCells {
			if tempCell.GetNumClients() > maxClientsInCell {
				toSplitIds = append(toSplitIds, tempCellId)
				toSplitCells = append(toSplitCells, tempCell)
			}

			// Add new cells
			splittingCell.AddChild(tempCellId, tempCell)
			r.addActiveCellToDeployment(deploymentId, tempCellId)

			// Remove old one
			splittingCell.Clear()
			r.removeActiveCellFromDeployment(deploymentId, splittingCellId)
		}
		splittingCell.Unlock()
	}
}

func (r *reqsLocationManager) addActiveCellToDeployment(deploymentId string, cellId s2.CellID) {
	value, ok := r.activeCells.Load(deploymentId)
	if !ok {
		return
	}

	activeCells := value.(activeCellsValue)
	activeCells.lock.Lock()
	activeCells.cells[cellId] = nil
	activeCells.lock.Unlock()
}

func (r *reqsLocationManager) removeActiveCellFromDeployment(deploymentId string, cellId s2.CellID) {
	value, ok := r.activeCells.Load(deploymentId)
	if !ok {
		return
	}

	activeCells := value.(activeCellsValue)
	activeCells.lock.Lock()
	delete(activeCells.cells, cellId)
	activeCells.lock.Unlock()
}

func (r *reqsLocationManager) manageLoadBatch() {
	ticker := time.NewTicker(batchTimer)

	for {
		<-ticker.C
		r.numReqsLock.Lock()
		for deploymentId, depBatch := range r.currBatch {
			go r.waitToRemove(deploymentId, depBatch)
			r.currBatch[deploymentId] = &batchValue{
				Locations: map[string]*locationsEntry{},
				NumReqs:   0,
			}
		}
		r.numReqsLock.Unlock()
	}
}

func (r *reqsLocationManager) waitToRemove(deploymentId string, entry *batchValue) {
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
}
