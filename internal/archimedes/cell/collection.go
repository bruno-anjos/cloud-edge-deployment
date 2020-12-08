package cell

import (
	"sync"

	"github.com/golang/geo/s2"
)

type (
	collection struct {
		cells *sync.Map
		*sync.RWMutex
	}
)

func newCollection() *collection {
	return &collection{
		cells:   &sync.Map{},
		RWMutex: &sync.RWMutex{},
	}
}

func (cc *collection) loadCell(cellId s2.CellID) (c *cell, loaded bool) {
	var value interface{}
	value, loaded = cc.cells.Load(cellId)
	if loaded {
		c = value.(*cell)
	}

	return
}

func (cc *collection) loadOrStoreCell(cellId s2.CellID, c *cell) (actual *cell, loaded bool) {
	var value interface{}
	value, loaded = cc.cells.LoadOrStore(cellId, c)
	return value.(*cell), loaded
}

func (cc *collection) storeCell(cellId s2.CellID, cell *cell) {
	cc.cells.Store(cellId, cell)
}

func (cc *collection) deleteCell(cellId s2.CellID) {
	cc.cells.Delete(cellId)
}

func (cc *collection) iterateCells(f func(id s2.CellID, cell *cell) bool) {
	cc.cells.Range(func(key, value interface{}) bool {
		return f(key.(s2.CellID), value.(*cell))
	})
}
