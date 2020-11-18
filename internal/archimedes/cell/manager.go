package cell

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/geo/s2"
	log "github.com/sirupsen/logrus"
)

type (
	LocationsMapKey   = s2.CellID
	LocationsMapValue = *int64

	cellsByDeploymentKey = string
	cellsByDeployment    struct {
		topCells *Collection
		cells    *Collection
	}

	cellsByDeploymentValue = *cellsByDeployment

	activeCellsKey               = s2.CellID
	activeCellsByDeploymentValue = *sync.Map

	Manager struct {
		cellsByDeployment sync.Map
		addDeploymentLock sync.Mutex
		activeCells       sync.Map
		splittedCells     sync.Map
	}
)

const (
	minCellLevel = 8
	maxCellLevel = 16

	maxClientsToSplit = 300
	minClientsToMerge = 200

	timeBetweenMerges = 30 * time.Second
)

func NewManager() *Manager {
	cm := &Manager{
		cellsByDeployment: sync.Map{},
		addDeploymentLock: sync.Mutex{},
		activeCells:       sync.Map{},
		splittedCells:     sync.Map{},
	}

	go cm.mergeCellsPeriodically()

	return cm
}

func (cm *Manager) GetDeploymentCentroids(deploymentId string) (centroids []s2.CellID, ok bool) {
	value, ok := cm.activeCells.Load(deploymentId)
	if !ok {
		return
	}

	deploymentActiveCells := value.(activeCellsByDeploymentValue)
	deploymentActiveCells.Range(func(key, _ interface{}) bool {
		id := key.(activeCellsKey)
		centroids = append(centroids, id)
		return true
	})

	return
}

func (cm *Manager) AddClientToDownmostCell(deploymentId string, clientCellId s2.CellID, logEntry *log.Entry) {
	deployment := cm.getDeploymentCells(deploymentId)
	topCellId, topCell := cm.getTopCell(deploymentId, deployment, clientCellId)

	downmostId, downmost := cm.findDownmostCellAndRLock(topCellId, topCell, clientCellId, deployment.cells, logEntry)
	logEntry.Debugf("add client to downmost cell %s", downmostId.ToToken())
	numClients := downmost.AddClientAndReturnCurrent(clientCellId)
	logEntry.Debug("RUnlocking deployment cells")
	deployment.cells.RUnlock()
	logEntry.Debug("RUnlocked")

	if numClients > maxCellLevel {
		_, loaded := cm.splittedCells.LoadOrStore(downmostId, nil)
		if !loaded {
			go cm.splitMaxedCell(deploymentId, deployment.cells, downmostId, downmost, logEntry)
		}
	}
}

func (cm *Manager) RemoveClientsFromCells(deploymentId string, locations *sync.Map) {
	deploymentCells := cm.getDeploymentCells(deploymentId)

	var (
		topCellId s2.CellID
		topCell   *Cell
	)

	locations.Range(func(key, value interface{}) bool {
		cellId := key.(LocationsMapKey)
		amount := value.(LocationsMapValue)

		topCellId, topCell = cm.getTopCell(deploymentId, deploymentCells, cellId)

		_, downmostCell := cm.findDownmostCellAndRLock(topCellId, topCell, cellId,
			deploymentCells.cells, log.WithField("REMOVING", true))

		downmostCell.RemoveClients(cellId, int(atomic.LoadInt64(amount)))
		deploymentCells.cells.RUnlock()

		return true
	})
}

func (cm *Manager) findDownmostCellAndRLock(topCellId s2.CellID, topCell *Cell, clientCellId s2.CellID,
	deploymentCells *Collection, logEntry *log.Entry) (downmostCellId s2.CellID, downmostCell *Cell) {
	level := topCellId.Level()
	downmostCellId = topCellId
	downmostCell = topCell

	logEntry.Debug("RLocking deployment cells")
	deploymentCells.RLock()
	logEntry.Debug("RLocked")

	for {
		if level == maxCellLevel {
			break
		}

		if len(downmostCell.Children) == 0 {
			break
		}

		for childId := range downmostCell.Children {
			if childId.Contains(clientCellId) {
				cell, ok := deploymentCells.LoadCell(childId)
				if !ok {
					logEntry.Fatalf("%s should have child %s", downmostCellId, childId)
				}
				downmostCellId = childId
				downmostCell = cell
				level++
				break
			}
		}

	}

	return
}

func (cm *Manager) getDeploymentCells(deploymentId string) (deployment *cellsByDeployment) {
	value, ok := cm.cellsByDeployment.Load(deploymentId)
	if !ok {
		cm.addDeploymentLock.Lock()
		defer cm.addDeploymentLock.Unlock()
		value, ok = cm.cellsByDeployment.Load(deploymentId)
		if !ok {
			deployment = &cellsByDeployment{
				topCells: newCollection(),
				cells:    newCollection(),
			}
			cm.cellsByDeployment.Store(deploymentId, deployment)
			return
		}
	}

	deployment = value.(cellsByDeploymentValue)

	return
}

func (cm *Manager) splitMaxedCell(deploymentId string, deploymentCells *Collection, cellId s2.CellID,
	cell *Cell, logEntry *log.Entry) {
	toSplitIds := []s2.CellID{cellId}
	toSplitCells := []*Cell{cell}

	logEntry.Debug("Locking deployment cells in SPLIT")
	deploymentCells.Lock()
	logEntry.Debug("Locked")
	defer func() {
		logEntry.Debug("Unlocking deployment cells in SPLIT")
		deploymentCells.Unlock()
		logEntry.Debug("Unlocked")
	}()

	var ok bool
	cell, ok = deploymentCells.LoadCell(cellId)
	if !ok || len(cell.Children) > 0 || cell.GetNumClientsNoLock() < maxClientsToSplit {
		logEntry.Debug("realised split was already done")
		return
	}

	for len(toSplitIds) > 0 {
		splittingCellId := toSplitIds[0]
		splittingCell := toSplitCells[0]

		toSplitIds = toSplitIds[1:]
		toSplitCells = toSplitCells[1:]

		logEntry.Debugf("splitting cell %d", splittingCellId)

		newCells := map[s2.CellID]*Cell{}

		splittingCell.IterateLocationsNoLock(func(locId s2.CellID, amount int) bool {
			newCellId := locId.Parent(splittingCellId.Level() + 1)
			tempCell, tempOk := newCells[newCellId]
			if !tempOk {
				newTempCell := NewCell(newCellId, amount, map[s2.CellID]int{locId: amount}, splittingCellId, true)
				newCells[newCellId] = newTempCell
			} else {
				tempCell.AddClientNoLock(locId)
			}
			return true
		})

		var activeCells *sync.Map
		activeCells, ok = cm.getActiveCellsForDeployment(deploymentId)
		if !ok {
			panic(fmt.Sprintf("should have active cells for deployment %s", deploymentId))
		}
		for tempCellId, tempCell := range newCells {
			if tempCell.GetNumClientsNoLock() > maxClientsToSplit {
				toSplitIds = append(toSplitIds, tempCellId)
				toSplitCells = append(toSplitCells, tempCell)
			}

			log.Debugf("added new cell %d to %d", tempCellId, splittingCellId)

			splittingCell.AddChild(tempCellId)
			activeCells.Store(tempCellId, tempCell)
		}
		splittingCell.ClearNoLock()
		activeCells.Delete(splittingCellId)
	}

}

func (cm *Manager) getActiveCellsForDeployment(deploymentId string) (*sync.Map, bool) {
	value, ok := cm.activeCells.Load(deploymentId)
	if !ok {
		return nil, false
	}
	return value.(activeCellsByDeploymentValue), ok
}

// Get one of the 4 top cells for a given deployment
func (cm *Manager) getTopCell(deploymentId string, deploymentCells cellsByDeploymentValue,
	clientCellId s2.CellID) (topCellId s2.CellID, topCell *Cell) {

	// search for the top cell that contains the client cell
	deploymentCells.topCells.IterateCells(func(id s2.CellID, cell *Cell) bool {
		if id.Contains(clientCellId) {
			topCellId = id
			topCell = cell
			return false
		}
		return true
	})

	if topCell == nil {
		// top cell didn't exist yet create it
		cellId := clientCellId.Parent(minCellLevel)
		cell := NewCell(cellId, 0, map[s2.CellID]int{}, 0, false)

		var loaded bool
		topCell, loaded = deploymentCells.topCells.LoadOrStoreCell(cellId, cell)

		// loadOrStore to sync map, so if it doens't load this thread is the one that created the cell
		if !loaded {
			// add the cell to activeCells
			var value activeCellsByDeploymentValue
			value = &sync.Map{}
			value.Store(cellId, cell)
			cm.activeCells.Store(deploymentId, value)
		}
	}

	return
}

func (cm *Manager) mergeCellsPeriodically() {
	wg := &sync.WaitGroup{}

	for {
		log.Debugf("merging cells")

		cm.cellsByDeployment.Range(func(key, value interface{}) bool {
			deploymentId := key.(cellsByDeploymentKey)
			deployment := value.(cellsByDeploymentValue)
			wg.Add(1)
			go cm.mergeDeploymentCells(deploymentId, deployment, wg)
			return true
		})

		wg.Wait()
		time.Sleep(timeBetweenMerges)
	}
}

func (cm *Manager) mergeDeploymentCells(deploymentId string, deployment *cellsByDeployment,
	topWg *sync.WaitGroup) {

	log.Debugf("Locking deployment cells in MERGE (%s)", deploymentId)
	deployment.cells.Lock()
	log.Debugf("Locked (%s)", deploymentId)

	wg := &sync.WaitGroup{}

	deployment.topCells.IterateCells(func(id s2.CellID, cell *Cell) bool {
		wg.Add(1)
		go cm.mergeFromTopCell(deploymentId, deployment.cells, cell, wg)
		return true
	})

	wg.Wait()

	log.Debugf("Unlocking deployment cells in MERGE (%s)", deploymentId)
	deployment.cells.Unlock()
	log.Debugf("Unlocked (%s)", deploymentId)

	topWg.Done()
}

func (cm *Manager) mergeFromTopCell(deploymentId string, deploymentCells *Collection, topCell *Cell,
	wg *sync.WaitGroup) {

	evaluateSet := cm.createEvaluateSet(topCell, deploymentCells)

	var (
		mergingCell       *Cell
		totalChildClients int
	)
	for i := len(evaluateSet) - 1; i >= 0; i-- {
		mergingCell = evaluateSet[i]

		totalChildClients = 0
		for childId := range mergingCell.Children {
			child, ok := deploymentCells.LoadCell(childId)
			if !ok {
				panic(fmt.Sprintf("has child %s, but child is not in deploymentCells", childId))
			}
			totalChildClients += child.GetNumClientsNoLock()
		}

		if totalChildClients < minClientsToMerge {
			activeCells, ok := cm.getActiveCellsForDeployment(deploymentId)
			if !ok {
				panic(fmt.Sprintf("should have active cells for deployment %s", deploymentId))
			}

			var child *Cell
			for childId := range mergingCell.Children {
				child, ok = deploymentCells.LoadCell(childId)
				if !ok {
					panic(fmt.Sprintf("has child %s, but child is not in deploymentCells", childId))
				}

				child.IterateLocationsNoLock(func(locId s2.CellID, amount int) bool {
					mergingCell.AddClientsNoLock(locId, amount)
					return true
				})

				deploymentCells.DeleteCell(childId)
				activeCells.Delete(childId)
			}

			cm.splittedCells.Delete(mergingCell.Id)
			mergingCell.Children = map[s2.CellID]interface{}{}
		}
	}

	wg.Done()
}

func (cm *Manager) createEvaluateSet(topCell *Cell, deploymentCells *Collection) (evaluateSet []*Cell) {
	if len(topCell.Children) == 0 {
		return
	}

	evaluateSet = append(evaluateSet, topCell)
	currentIdx := 0

	for currentIdx < len(evaluateSet) {
		toExplore := evaluateSet[currentIdx]

		for childId := range toExplore.Children {
			child, ok := deploymentCells.LoadCell(childId)
			if !ok {
				panic(fmt.Sprintf("should have cell %d", childId))
			}

			if len(child.Children) > 0 {
				evaluateSet = append(evaluateSet, child)
			}
		}

		currentIdx++
	}

	return
}
