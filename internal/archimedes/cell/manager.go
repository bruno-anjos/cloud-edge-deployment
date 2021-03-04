package cell

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/environment"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/golang/geo/s2"
	client "github.com/nm-morais/demmon-client/pkg"
	log "github.com/sirupsen/logrus"
)

type (
	LocationsMapKey   = s2.CellID
	LocationsMapValue = *int64

	cellsByDeploymentKey = string
	cellsByDeployment    struct {
		topCells *collection
		cells    *collection
	}

	cellsByDeploymentValue = *cellsByDeployment

	activeCellsKey               = s2.CellID
	activeCellsByDeploymentValue = *sync.Map

	Manager struct {
		cellsByDeployment sync.Map
		addDeploymentLock sync.Mutex
		activeCells       sync.Map
		splittedCells     sync.Map
		demmCli           *client.DemmonClient
		myself            *utils.Node
	}
)

const (
	minCellLevel = 8
	maxCellLevel = 18

	maxClientsToSplit = 300
	minClientsToMerge = 200

	timeBetweenMerges = 30 * time.Second
)

func NewManager(demmCli *client.DemmonClient, myself *utils.Node) *Manager {
	cm := &Manager{
		cellsByDeployment: sync.Map{},
		addDeploymentLock: sync.Mutex{},
		activeCells:       sync.Map{},
		splittedCells:     sync.Map{},
		demmCli:           demmCli,
		myself:            myself,
	}

	environment.SetupClientCentroidsExport(demmCli)

	go cm.mergeCellsPeriodically()

	return cm
}

func (cm *Manager) GetDeploymentCentroids(deploymentID string) (centroids []s2.CellID, ok bool) {
	value, ok := cm.activeCells.Load(deploymentID)
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

func (cm *Manager) AddClientToDownmostCell(deploymentID string, clientCellID s2.CellID) {
	log.Debugf("%s adding cell %s", deploymentID, clientCellID.ToToken())

	deployment := cm.getDeploymentCells(deploymentID)
	topCellID, topCell := cm.getTopCell(deploymentID, deployment, clientCellID)

	downmostID, downmost := cm.findDownmostCellAndRLock(topCellID, topCell, clientCellID, deployment.cells)
	numClients := downmost.addClientAndReturnCurrent(clientCellID)
	deployment.cells.RUnlock()

	if numClients > maxClientsToSplit {
		_, loaded := cm.splittedCells.LoadOrStore(downmostID, nil)
		if !loaded {
			go cm.splitMaxedCell(deploymentID, deployment.cells, downmostID, downmost)
		}
	}
}

func (cm *Manager) RemoveClientsFromCells(deploymentID string, locations *sync.Map) {
	deploymentCells := cm.getDeploymentCells(deploymentID)

	var (
		topCellID s2.CellID
		topCell   *cell
	)

	locations.Range(func(key, value interface{}) bool {
		cellID := key.(LocationsMapKey)
		amount := value.(LocationsMapValue)

		topCellID, topCell = cm.getTopCell(deploymentID, deploymentCells, cellID)

		_, downmostCell := cm.findDownmostCellAndRLock(topCellID, topCell, cellID, deploymentCells.cells)

		downmostCell.removeClients(cellID, int(atomic.LoadInt64(amount)))
		deploymentCells.cells.RUnlock()

		return true
	})
}

func (cm *Manager) findDownmostCellAndRLock(topCellID s2.CellID, topCell *cell, clientCellID s2.CellID,
	deploymentCells *collection) (downmostCellID s2.CellID, downmostCell *cell) {
	level := topCellID.Level()
	downmostCellID = topCellID
	downmostCell = topCell

	deploymentCells.RLock()

	for {
		if level == maxCellLevel {
			break
		}

		if len(downmostCell.Children) == 0 {
			break
		}

		for childID := range downmostCell.Children {
			if childID.Contains(clientCellID) {
				c, ok := deploymentCells.loadCell(childID)
				if !ok {
					log.Panicf("%s should have child %s", downmostCellID, childID)
				}

				downmostCellID = childID
				downmostCell = c
				level++

				break
			}
		}
	}

	return downmostCellID, downmostCell
}

func (cm *Manager) getDeploymentCells(deploymentID string) (deployment *cellsByDeployment) {
	value, ok := cm.cellsByDeployment.Load(deploymentID)
	if !ok {
		cm.addDeploymentLock.Lock()
		defer cm.addDeploymentLock.Unlock()

		value, ok = cm.cellsByDeployment.Load(deploymentID)
		if !ok {
			deployment = &cellsByDeployment{
				topCells: newCollection(),
				cells:    newCollection(),
			}
			cm.cellsByDeployment.Store(deploymentID, deployment)

			return
		}
	}

	deployment = value.(cellsByDeploymentValue)

	return
}

func (cm *Manager) splitMaxedCell(deploymentID string, deploymentCells *collection, cellID s2.CellID, c *cell) {
	toSplitIds := []s2.CellID{cellID}
	toSplitCells := []*cell{c}

	deploymentCells.Lock()
	defer func() {
		deploymentCells.Unlock()
	}()

	var ok bool

	c, ok = deploymentCells.loadCell(cellID)
	if !ok || len(c.Children) > 0 || c.getNumClientsNoLock() < maxClientsToSplit {
		return
	}

	for len(toSplitIds) > 0 {
		splittingCellID := toSplitIds[0]
		splittingCell := toSplitCells[0]

		toSplitIds = toSplitIds[1:]
		toSplitCells = toSplitCells[1:]

		log.Debugf("splitting cell %d", splittingCellID)

		newCells := map[s2.CellID]*cell{}

		splittingCell.iterateLocationsNoLock(func(locId s2.CellID, amount int) bool {
			newCellID := locId.Parent(splittingCellID.Level() + 1)
			tempCell, tempOk := newCells[newCellID]
			if !tempOk {
				newTempCell := newCell(newCellID, amount, map[s2.CellID]int{locId: amount}, splittingCellID, true)
				newCells[newCellID] = newTempCell
			} else {
				tempCell.addClientNoLock(locId)
			}

			return true
		})

		var activeCells *sync.Map

		activeCells, ok = cm.getActiveCellsForDeployment(deploymentID)
		if !ok {
			log.Panic(fmt.Sprintf("should have active cells for deployment %s", deploymentID))
		}

		for tempCellID, tempCell := range newCells {
			if tempCell.getNumClientsNoLock() > maxClientsToSplit {
				toSplitIds = append(toSplitIds, tempCellID)
				toSplitCells = append(toSplitCells, tempCell)
			}

			log.Debugf("added new cell %d to %d", tempCellID, splittingCellID)

			splittingCell.addChild(tempCellID)
			activeCells.Store(tempCellID, tempCell)
		}

		splittingCell.clearNoLock()
		activeCells.Delete(splittingCellID)
	}
}

func (cm *Manager) getActiveCellsForDeployment(deploymentID string) (*sync.Map, bool) {
	value, ok := cm.activeCells.Load(deploymentID)
	if !ok {
		return nil, false
	}

	return value.(activeCellsByDeploymentValue), ok
}

// Get one of the 4 top cells for a given deployment.
func (cm *Manager) getTopCell(deploymentID string, deploymentCells cellsByDeploymentValue,
	clientCellID s2.CellID) (topCellID s2.CellID, topCell *cell) {
	// search for the top cell that contains the client cell
	deploymentCells.topCells.iterateCells(func(id s2.CellID, cell *cell) bool {
		if id.Contains(clientCellID) {
			topCellID = id
			topCell = cell

			return false
		}

		return true
	})

	if topCell == nil {
		// top cell didn't exist yet create it
		cellID := clientCellID.Parent(minCellLevel)
		c := newCell(cellID, 0, map[s2.CellID]int{}, 0, false)

		var loaded bool
		topCell, loaded = deploymentCells.topCells.loadOrStoreCell(cellID, c)

		// loadOrStore to sync map, so if it doesn't load, this thread is the one that created the cell
		if !loaded {
			// add the cell to activeCells
			cells := &sync.Map{}
			cells.Store(cellID, c)

			var value interface{}

			value, loaded = cm.activeCells.LoadOrStore(deploymentID, cells)
			if loaded {
				value.(*sync.Map).Store(cellID, c)
			}

			var centroids []s2.CellID
			value.(*sync.Map).Range(func(key, _ interface{}) bool {
				id := key.(activeCellsKey)
				centroids = append(centroids, id)

				return true
			})

			environment.ExportClientCentroids(cm.demmCli, deploymentID, cm.myself, centroids)
		}
	}

	return topCellID, topCell
}

func (cm *Manager) mergeCellsPeriodically() {
	wg := &sync.WaitGroup{}

	for {
		log.Debugf("merging cells")

		cm.cellsByDeployment.Range(func(key, value interface{}) bool {
			deploymentID := key.(cellsByDeploymentKey)
			deployment := value.(cellsByDeploymentValue)
			wg.Add(1)
			go cm.mergeDeploymentCells(deploymentID, deployment, wg)

			return true
		})

		wg.Wait()
		time.Sleep(timeBetweenMerges)
	}
}

func (cm *Manager) mergeDeploymentCells(deploymentID string, deployment *cellsByDeployment,
	topWg *sync.WaitGroup) {
	log.Debugf("Locking deployment cells in MERGE (%s)", deploymentID)
	deployment.cells.Lock()
	log.Debugf("Locked (%s)", deploymentID)

	wg := &sync.WaitGroup{}

	deployment.topCells.iterateCells(func(id s2.CellID, cell *cell) bool {
		wg.Add(1)
		go cm.mergeFromTopCell(deploymentID, deployment.cells, cell, wg)

		return true
	})

	wg.Wait()

	log.Debugf("Unlocking deployment cells in MERGE (%s)", deploymentID)
	deployment.cells.Unlock()
	log.Debugf("Unlocked (%s)", deploymentID)

	topWg.Done()
}

func (cm *Manager) mergeFromTopCell(deploymentID string, deploymentCells *collection, topCell *cell,
	wg *sync.WaitGroup) {
	evaluateSet := cm.createEvaluateSet(topCell, deploymentCells)

	var (
		mergingCell       *cell
		totalChildClients int
	)

	for i := len(evaluateSet) - 1; i >= 0; i-- {
		mergingCell = evaluateSet[i]

		totalChildClients = 0

		for childID := range mergingCell.Children {
			child, ok := deploymentCells.loadCell(childID)
			if !ok {
				log.Panic(fmt.Sprintf("has child %s, but child is not in deploymentCells", childID))
			}

			totalChildClients += child.getNumClientsNoLock()
		}

		if totalChildClients < minClientsToMerge {
			activeCells, ok := cm.getActiveCellsForDeployment(deploymentID)
			if !ok {
				log.Panic(fmt.Sprintf("should have active cells for deployment %s", deploymentID))
			}

			var child *cell
			for childID := range mergingCell.Children {
				child, ok = deploymentCells.loadCell(childID)
				if !ok {
					log.Panic(fmt.Sprintf("has child %s, but child is not in deploymentCells", childID))
				}

				child.iterateLocationsNoLock(func(locId s2.CellID, amount int) bool {
					mergingCell.addClientsNoLock(locId, amount)

					return true
				})

				deploymentCells.deleteCell(childID)
				activeCells.Delete(childID)
			}

			cm.splittedCells.Delete(mergingCell.ID)
			mergingCell.Children = map[s2.CellID]interface{}{}
		}
	}

	wg.Done()
}

func (cm *Manager) createEvaluateSet(topCell *cell, deploymentCells *collection) (evaluateSet []*cell) {
	if len(topCell.Children) == 0 {
		return
	}

	evaluateSet = append(evaluateSet, topCell)
	currentIdx := 0

	for currentIdx < len(evaluateSet) {
		toExplore := evaluateSet[currentIdx]

		for childID := range toExplore.Children {
			child, ok := deploymentCells.loadCell(childID)
			if !ok {
				log.Panic(fmt.Sprintf("should have cell %d", childID))
			}

			if len(child.Children) > 0 {
				evaluateSet = append(evaluateSet, child)
			}
		}

		currentIdx++
	}

	return
}
