package cell

import (
	"sync"

	"github.com/golang/geo/s2"
)

type (
	Cell struct {
		numClients      int
		clientLocations map[s2.CellID]int
		HasParent       bool
		Parent          s2.CellID
		Children        map[s2.CellID]interface{}
		sync.RWMutex
	}
)

func NewCell(numClients int, clientLocations map[s2.CellID]int, parent s2.CellID, hasParent bool) *Cell {
	return &Cell{
		numClients:      numClients,
		clientLocations: clientLocations,
		Children:        map[s2.CellID]interface{}{},
		Parent:          parent,
		HasParent:       hasParent,
	}
}

func (c *Cell) AddClientAndReturnCurrent(loc s2.CellID) int {
	c.Lock()
	defer c.Unlock()
	c.AddClientNoLock(loc)
	return c.numClients
}

func (c *Cell) AddClientNoLock(loc s2.CellID) {
	c.numClients++
	c.clientLocations[loc]++
}

func (c *Cell) AddClientsNoLock(loc s2.CellID, amount int) {
	c.numClients += amount
	c.clientLocations[loc] += amount
}

func (c *Cell) RemoveClient(loc s2.CellID) {
	c.Lock()
	defer c.Unlock()
	c.RemoveClientNoLock(loc)
}

func (c *Cell) RemoveClients(loc s2.CellID, amount int) {
	c.Lock()
	defer c.Unlock()
	_, ok := c.clientLocations[loc]
	if !ok {
		return
	}

	c.numClients -= amount
	c.clientLocations[loc] -= amount
	if c.clientLocations[loc] == 0 {
		delete(c.clientLocations, loc)
	}
}

func (c *Cell) RemoveClientsNoLock(loc s2.CellID, amount int) {
	c.numClients -= amount
	c.clientLocations[loc]--
	if c.clientLocations[loc] == 0 {
		delete(c.clientLocations, loc)
	}
}

func (c *Cell) RemoveClientNoLock(loc s2.CellID) {
	c.numClients--
	c.clientLocations[loc]--
	if c.clientLocations[loc] == 0 {
		delete(c.clientLocations, loc)
	}
}

func (c *Cell) GetNumClients() int {
	c.RLock()
	defer c.RUnlock()
	return c.GetNumClientsNoLock()
}

func (c *Cell) GetNumClientsNoLock() int {
	return c.numClients
}

func (c *Cell) IterateLocationsNoLock(f func(locId s2.CellID, amount int) bool) {
	for locId, amount := range c.clientLocations {
		if !f(locId, amount) {
			break
		}
	}
}

func (c *Cell) ClearNoLock() {
	c.numClients = 0
	c.clientLocations = nil
	c.Children = map[s2.CellID]interface{}{}
}

func (c *Cell) AddChild(childId s2.CellID) {
	c.Children[childId] = nil
}

func (c *Cell) DeleteChild(childId s2.CellID) {
	delete(c.Children, childId)
}

func (c *Cell) HasChild(childId s2.CellID) bool {
	_, ok := c.Children[childId]
	return ok
}
