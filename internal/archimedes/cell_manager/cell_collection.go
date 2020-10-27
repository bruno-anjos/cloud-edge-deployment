package cell_manager

import (
	"sync"

	"github.com/golang/geo/s2"
)

type (
	CellsCollection struct {
		cells map[s2.CellID]*Cell
		sync.RWMutex
	}
)

func (cc *CellsCollection) LoadOrStoreCell(cellId s2.CellID, cell *Cell) (actual *Cell, loaded bool) {
	cc.RWMutex.RLock()
	actual, loaded = cc.cells[cellId]
	if loaded {
		cc.RWMutex.RUnlock()
		return
	}
	cc.RWMutex.RUnlock()

	cc.RWMutex.Lock()
	defer cc.RWMutex.Unlock()
	actual, loaded = cc.cells[cellId]
	if loaded {
		return
	}

	cc.cells[cellId] = cell
	actual = cell

	return
}

func (cc *CellsCollection) DeleteCell(cellId s2.CellID) {
	cc.Lock()
	delete(cc.cells, cellId)
	cc.Unlock()
}

func (cc *CellsCollection) IterateCellsNoLock(f func(id s2.CellID, cell *Cell) bool) {
	for cellId, cell := range cc.cells {
		if !f(cellId, cell) {
			break
		}
	}
}
