package cell

import (
	"sync"

	"github.com/golang/geo/s2"
)

type (
	cell struct {
		Id              s2.CellID
		numClients      int
		clientLocations map[s2.CellID]int
		HasParent       bool
		Parent          s2.CellID
		Children        map[s2.CellID]interface{}
		*sync.RWMutex
	}
)

func newCell(id s2.CellID, numClients int, clientLocations map[s2.CellID]int, parent s2.CellID, hasParent bool) *cell {
	return &cell{
		Id:              id,
		numClients:      numClients,
		clientLocations: clientLocations,
		Children:        map[s2.CellID]interface{}{},
		Parent:          parent,
		HasParent:       hasParent,
		RWMutex:         &sync.RWMutex{},
	}
}

func (c *cell) addClientAndReturnCurrent(loc s2.CellID) int {
	c.Lock()
	defer c.Unlock()
	c.addClientNoLock(loc)
	return c.numClients
}

func (c *cell) addClientNoLock(loc s2.CellID) {
	c.numClients++
	c.clientLocations[loc]++
}

func (c *cell) addClientsNoLock(loc s2.CellID, amount int) {
	c.numClients += amount
	c.clientLocations[loc] += amount
}

func (c *cell) removeClient(loc s2.CellID) {
	c.Lock()
	defer c.Unlock()
	c.removeClientNoLock(loc)
}

func (c *cell) removeClients(loc s2.CellID, amount int) {
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

func (c *cell) removeClientsNoLock(loc s2.CellID, amount int) {
	c.numClients -= amount
	c.clientLocations[loc]--
	if c.clientLocations[loc] == 0 {
		delete(c.clientLocations, loc)
	}
}

func (c *cell) removeClientNoLock(loc s2.CellID) {
	c.numClients--
	c.clientLocations[loc]--
	if c.clientLocations[loc] == 0 {
		delete(c.clientLocations, loc)
	}
}

func (c *cell) getNumClients() int {
	c.RLock()
	defer c.RUnlock()
	return c.getNumClientsNoLock()
}

func (c *cell) getNumClientsNoLock() int {
	return c.numClients
}

func (c *cell) iterateLocationsNoLock(f func(locId s2.CellID, amount int) bool) {
	for locId, amount := range c.clientLocations {
		if !f(locId, amount) {
			break
		}
	}
}

func (c *cell) clearNoLock() {
	c.numClients = 0
	c.clientLocations = nil
	c.Children = map[s2.CellID]interface{}{}
}

func (c *cell) addChild(childId s2.CellID) {
	c.Children[childId] = nil
}

func (c *cell) deleteChild(childId s2.CellID) {
	delete(c.Children, childId)
}

func (c *cell) hasChild(childId s2.CellID) bool {
	_, ok := c.Children[childId]
	return ok
}
