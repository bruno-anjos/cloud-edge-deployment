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

func (cc *collection) loadCell(cellID s2.CellID) (c *cell, loaded bool) {
	var value interface{}

	value, loaded = cc.cells.Load(cellID)
	if loaded {
		c = value.(*cell)
	}

	return
}

func (cc *collection) loadOrStoreCell(cellID s2.CellID, c *cell) (actual *cell, loaded bool) {
	var value interface{}
	value, loaded = cc.cells.LoadOrStore(cellID, c)

	return value.(*cell), loaded
}

func (cc *collection) deleteCell(cellID s2.CellID) {
	cc.cells.Delete(cellID)
}

func (cc *collection) iterateCells(f func(id s2.CellID, cell *cell) bool) {
	cc.cells.Range(func(key, value interface{}) bool {
		return f(key.(s2.CellID), value.(*cell))
	})
}
