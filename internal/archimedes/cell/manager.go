package cell

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/golang/geo/s2"
)

type (
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

func (cm *Manager) AddClientToDownmostCell(deploymentId string, clientCellId s2.CellID) {
	deployment := cm.getDeploymentCells(deploymentId)
	topCellId, topCell := cm.getTopCell(deploymentId, deployment, clientCellId)

	downmostId, downmost := cm.findDownmostCellAndRLock(topCellId, topCell, clientCellId, deployment.cells)
	numClients := downmost.AddClientAndReturnCurrent(clientCellId)
	deployment.cells.RUnlock()

	if numClients > maxCellLevel {
		go cm.splitMaxedCell(deploymentId, deployment.cells, downmostId, downmost)
	}
}

func (cm *Manager) RemoveClientsFromCells(deploymentId string, locations map[s2.CellID]int) {
	deploymentCells := cm.getDeploymentCells(deploymentId)

	var (
		topCellId s2.CellID
		topCell   *Cell
	)
	for cellId, amount := range locations {
		topCellId, topCell = cm.getTopCell(deploymentId, deploymentCells, cellId)

		_, downmostCell := cm.findDownmostCellAndRLock(topCellId, topCell, cellId,
			deploymentCells.cells)

		downmostCell.RemoveClients(cellId, amount)
	}
}

func (cm *Manager) findDownmostCellAndRLock(topCellId s2.CellID, topCell *Cell, clientCellId s2.CellID,
	deploymentCells *Collection) (downmostCellId s2.CellID, downmostCell *Cell) {
	level := topCellId.Level()
	downmostCellId = topCellId
	downmostCell = topCell

	deploymentCells.RLock()

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
					log.Fatalf("%s should have child %s", downmostCellId, childId)
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
		value, ok = cm.cellsByDeployment.Load(deploymentId)
		if !ok {
			deployment = &cellsByDeployment{
				topCells: newCollection(),
				cells:    newCollection(),
			}
			cm.cellsByDeployment.Store(deploymentId, deployment)
			return
		}
		cm.addDeploymentLock.Unlock()
	}

	deployment = value.(cellsByDeploymentValue)

	return
}

func (cm *Manager) splitMaxedCell(deploymentId string, deploymentCells *Collection, cellId s2.CellID,
	cell *Cell) {
	toSplitIds := []s2.CellID{cellId}
	toSplitCells := []*Cell{cell}

	deploymentCells.Lock()
	defer deploymentCells.Unlock()

	var ok bool
	cell, ok = deploymentCells.LoadCell(cellId)
	if !ok || len(cell.Children) > 0 || cell.GetNumClientsNoLock() < maxClientsToSplit {
		return
	}

	for len(toSplitIds) > 0 {
		splittingCellId := toSplitIds[0]
		splittingCell := toSplitCells[0]

		toSplitIds = toSplitIds[1:]
		toSplitCells = toSplitCells[1:]

		newCells := map[s2.CellID]*Cell{}

		splittingCell.IterateLocationsNoLock(func(locId s2.CellID, amount int) bool {
			newCellId := locId.Parent(splittingCellId.Level() + 1)
			tempCell, tempOk := newCells[newCellId]
			if !tempOk {
				newTempCell := NewCell(amount, map[s2.CellID]int{locId: amount}, splittingCellId, true)
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

func (cm *Manager) getTopCell(deploymentId string, deploymentCells cellsByDeploymentValue,
	clientCellId s2.CellID) (topCellId s2.CellID, topCell *Cell) {

	deploymentCells.topCells.IterateCells(func(id s2.CellID, cell *Cell) bool {
		if id.Contains(clientCellId) {
			topCellId = id
			topCell = cell
			return false
		}
		return true
	})

	if topCell == nil {
		cellId := clientCellId.Parent(minCellLevel)
		cell := NewCell(0, map[s2.CellID]int{}, 0, false)

		var loaded bool
		topCell, loaded = deploymentCells.topCells.LoadOrStoreCell(cellId, cell)

		if !loaded {
			activeCells, ok := cm.getActiveCellsForDeployment(deploymentId)
			if !ok {
				panic(fmt.Sprintf("should have active cells for deployment %s", deploymentId))
			}
			activeCells.Store(cellId, cell)
		}
	}

	return
}

func (cm *Manager) mergeCellsPeriodically() {
	wg := &sync.WaitGroup{}

	for {
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

	deployment.cells.Lock()
	wg := &sync.WaitGroup{}

	deployment.topCells.IterateCells(func(id s2.CellID, cell *Cell) bool {
		wg.Add(1)
		go cm.mergeFromTopCell(deploymentId, deployment.cells, cell, wg)
		return true
	})

	wg.Wait()
	deployment.cells.Unlock()

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
