package cell

import (
	"sync"

	"github.com/golang/geo/s2"
)

type (
	Collection struct {
		cells *sync.Map
		sync.RWMutex
	}
)

func newCollection() *Collection {
	return &Collection{
		cells:   &sync.Map{},
		RWMutex: sync.RWMutex{},
	}
}

func (cc *Collection) LoadCell(cellId s2.CellID) (cell *Cell, loaded bool) {
	var value interface{}
	value, loaded = cc.cells.Load(cellId)
	return value.(*Cell), loaded
}

func (cc *Collection) LoadOrStoreCell(cellId s2.CellID, cell *Cell) (actual *Cell, loaded bool) {
	var value interface{}
	value, loaded = cc.cells.LoadOrStore(cellId, cell)
	return value.(*Cell), loaded
}

func (cc *Collection) StoreCell(cellId s2.CellID, cell *Cell) {
	cc.cells.Store(cellId, cell)
}

func (cc *Collection) DeleteCell(cellId s2.CellID) {
	cc.cells.Delete(cellId)
}

func (cc *Collection) IterateCells(f func(id s2.CellID, cell *Cell) bool) {
	cc.cells.Range(func(key, value interface{}) bool {
		return f(key.(s2.CellID), value.(*Cell))
	})
}
