package cell_manager

import (
	"github.com/golang/geo/s2"
)

type (
	CellsCollection struct {
		cells map[s2.CellID]*Cell
	}
)

func (cc *CellsCollection) AddCell(cellId s2.CellID, cell *Cell) {
	cc.cells[cellId] = cell
}

func (cc *CellsCollection) RemoveCell(cellId s2.CellID) {
	delete(cc.cells, cellId)
}

func (cc *CellsCollection) Iterate(f func(id s2.CellID, cell *Cell) bool) {
	for id, cell := range cc.cells {
		if !f(id, cell) {
			break
		}
	}
}