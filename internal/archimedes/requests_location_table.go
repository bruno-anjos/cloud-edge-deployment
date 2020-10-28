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
		cells map[s2.CellID]*cell_manager.Cell
		sync.RWMutex
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

	maxClientsToSplit = 300
	minClientsToMerge = 200

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

	r.addClientToCells(deploymentId, location)
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

	deploymentCells.RLock()
	deploymentCells.IterateCellsNoLock(func(id s2.CellID, cell *cell_manager.Cell) bool {
		hasLocations = true
		cellCenter := s2.LatLngFromPoint(s2.CellFromCellID(id).Center())
		centroids = append(centroids, publicUtils.FromLatLngToLocation(&cellCenter))
		return true
	})
	deploymentCells.RUnlock()
	ok = hasLocations

	return
}

func (r *reqsLocationManager) addToExploring(deploymentId string, cells []s2.CellID) {
	r.exploringCells.Store(deploymentId, cells)
}

func (r *reqsLocationManager) removeFromExploring(deploymentId string) {
	r.exploringCells.Delete(deploymentId)
}

func (r *reqsLocationManager) addClientToCells(deploymentId string, location *publicUtils.Location) {
	deploymentCells := r.getDeploymentCells(deploymentId)

	leafId := s2.CellIDFromLatLng(location.ToLatLng())

	missingTopCell := true
	deploymentCells.RLock()
	deploymentCells.IterateCellsNoLock(func(id s2.CellID, cell *cell_manager.Cell) bool {
		if id.Contains(leafId) {
			r.addToDownmostCell(id, cell, leafId, deploymentId, location)
			missingTopCell = false
			return false
		}
		return true
	})
	deploymentCells.RUnlock()

	if missingTopCell {
		cell := cell_manager.NewCell(0, map[string]*cell_manager.ClientLocation{})
		cellId := s2.CellIDFromLatLng(location.ToLatLng()).Parent(minCellLevel)

		var loaded bool
		cell, loaded = deploymentCells.LoadOrStoreCell(cellId, cell)
		if !loaded {
			r.activeCells.Store(cellId, cell)
		}

		cell.AddClient(location)
	}
}

func (r *reqsLocationManager) removeClientsFromCells(deploymentId string, locations map[string]*locationsEntry) {
	deploymentCells := r.getDeploymentCells(deploymentId)

	var (
		topCellId s2.CellID
		topCell   *cell_manager.Cell
	)
	for _, locEntry := range locations {
		cellId := s2.CellIDFromLatLng(locEntry.Location.ToLatLng())

		deploymentCells.RLock()
		deploymentCells.IterateCellsNoLock(func(id s2.CellID, cell *cell_manager.Cell) bool {
			if id.Contains(cellId) {
				topCellId = id
				topCell = cell
				return false
			}
			return true
		})
		deploymentCells.RUnlock()

		downmostCellId, downmostCell, locks, level := r.findDownmostCellAndLock(topCellId, topCell, cellId)
		downmostCell.RemoveClientsNoLock(locEntry.Location, locEntry.Number)

		possibleChildIds := downmostCellId.Parent(level).Children()
		totalNumClients := 0
		for _, childId := range possibleChildIds {
			cell, ok := r.getActiveCellFromDeployment(deploymentId, childId)
			if ok {
				totalNumClients += cell.GetNumClients()
			}
		}

		if totalNumClients < minClientsToMerge {
			go mergeCells()
		}

		downmostCell.Unlock()

		for i := range locks {
			locks[len(locks)-1-i].RUnlock()
		}
	}
}

func (r *reqsLocationManager) findDownmostCellAndLock(cellId s2.CellID, c *cell_manager.Cell,
	leafId s2.CellID) (s2.CellID, *cell_manager.Cell, []*sync.RWMutex, int) {
	level := cellId.Level()
	discoverCell := c
	resultCellId := cellId
	var locks []*sync.RWMutex

searchChild:
	for {
		hasChildren := false
		childContains := false

		discoverCell.RLock()

		var nextCell *cell_manager.Cell
		c.Children.RLock()
		c.Children.IterateCellsNoLock(func(id s2.CellID, cell *cell_manager.Cell) bool {
			hasChildren = true
			if id.Contains(leafId) {
				childContains = true
				nextCell = cell
				resultCellId = id
				level++
				return false
			}
			return true
		})

		if !hasChildren || level > maxCellLevel {
			c.Children.RUnlock()
			break
		}

		if !childContains {
			c.Children.RUnlock()
			level++
			resultCellId = leafId.Parent(level)
			discoverCell = cell_manager.NewCell(0, map[string]*cell_manager.ClientLocation{})

			var loaded bool
			discoverCell, loaded = c.Children.LoadOrStoreCell(resultCellId, discoverCell)
			if !loaded {
				r.activeCells.Store(resultCellId, discoverCell)
			}

			break
		}

		discoverCell.RUnlock()
		discoverCell = nextCell

		locks = append(locks, &c.Children.RWMutex)
	}

	hasChildren := false

	discoverCell.Lock()

	discoverCell.Children.RLock()
	discoverCell.Children.IterateCellsNoLock(func(id s2.CellID, cell *cell_manager.Cell) bool {
		hasChildren = true
		return false
	})
	discoverCell.Children.RUnlock()

	if hasChildren {
		discoverCell.Unlock()
		goto searchChild
	}

	return resultCellId, discoverCell, locks, level
}

func (r *reqsLocationManager) addToDownmostCell(cellId s2.CellID, c *cell_manager.Cell, leafId s2.CellID,
	deploymentId string, location *publicUtils.Location) {

	resultCellId, discoverCell, locks, _ := r.findDownmostCellAndLock(cellId, c, leafId)
	discoverCell.AddClientNoLock(location)

	if resultCellId.Level() < maxCellLevel && discoverCell.GetNumClientsNoLock() == maxClientsToSplit {
		go r.splitMaxedCell(deploymentId, resultCellId, discoverCell)
	}

	discoverCell.Unlock()

	for i := range locks {
		locks[len(locks)-1-i].RUnlock()
	}
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

		splittingCell.IterateLocationsNoLock(func(locId string, loc *cell_manager.ClientLocation) bool {
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
			if tempCell.GetNumClients() > maxClientsToSplit {
				toSplitIds = append(toSplitIds, tempCellId)
				toSplitCells = append(toSplitCells, tempCell)
			}

			// Add new cells
			var loaded bool
			tempCell, loaded = splittingCell.Children.LoadOrStoreCell(tempCellId, tempCell)
			if !loaded {
				r.addActiveCellToDeployment(deploymentId, tempCellId)
			}

			// Remove old one
			splittingCell.ClearNoLock()
			r.removeActiveCellFromDeployment(deploymentId, splittingCellId)
		}
		splittingCell.Unlock()
	}
}

func (r *reqsLocationManager) addActiveCellToDeployment(deploymentId string, cellId s2.CellID, cell *cell_manager.Cell) {
	value, ok := r.activeCells.Load(deploymentId)
	if !ok {
		return
	}

	activeCells := value.(activeCellsValue)
	activeCells.Lock()
	activeCells.cells[cellId] = cell
	activeCells.Unlock()
}

func (r *reqsLocationManager) removeActiveCellFromDeployment(deploymentId string, cellId s2.CellID) {
	value, ok := r.activeCells.Load(deploymentId)
	if !ok {
		return
	}

	activeCells := value.(activeCellsValue)
	activeCells.Lock()
	delete(activeCells.cells, cellId)
	activeCells.Unlock()
}

func (r *reqsLocationManager) getActiveCellFromDeployment(deploymentId string, cellId s2.CellID) (*cell_manager.Cell,
	bool) {
	value, ok := r.activeCells.Load(deploymentId)
	if !ok {
		return nil, false
	}

	activeCells := value.(activeCellsValue)
	activeCells.RLock()
	cell, ok := activeCells.cells[cellId]
	activeCells.RUnlock()

	return cell, ok
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

	r.removeClientFromCells(deploymentId, entry.Locations)
}
