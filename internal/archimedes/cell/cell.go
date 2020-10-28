package cell

import (
	"sync"

	publicUtils "github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/golang/geo/s2"
)

type (
	ClientLocation struct {
		Location *publicUtils.Location
		Count    int
	}

	Cell struct {
		numClients      int
		clientLocations map[string]*ClientLocation
		HasParent       bool
		Parent          s2.CellID
		Children        map[s2.CellID]interface{}
		sync.RWMutex
	}
)

func NewCell(numClients int, clientLocations map[string]*ClientLocation, parent s2.CellID, hasParent bool) *Cell {
	return &Cell{
		numClients:      numClients,
		clientLocations: clientLocations,
		Children:        map[s2.CellID]interface{}{},
		Parent:          parent,
		HasParent:       hasParent,
	}
}

func (c *Cell) AddClientAndReturnCurrent(loc *publicUtils.Location) int {
	c.Lock()
	defer c.Unlock()
	c.AddClientNoLock(loc)
	return c.numClients
}

func (c *Cell) AddClientNoLock(loc *publicUtils.Location) {
	c.numClients++
	id := loc.GetId()
	value, ok := c.clientLocations[id]
	if !ok {
		c.clientLocations[id] = &ClientLocation{
			Location: loc,
			Count:    1,
		}
	} else {
		value.Count++
	}
}

func (c *Cell) AddClientsNoLock(loc *publicUtils.Location, amount int) {
	c.numClients += amount
	id := loc.GetId()
	value, ok := c.clientLocations[id]
	if !ok {
		c.clientLocations[id] = &ClientLocation{
			Location: loc,
			Count:    amount,
		}
	} else {
		value.Count += amount
	}
}

func (c *Cell) RemoveClient(loc *publicUtils.Location) {
	c.Lock()
	defer c.Unlock()
	c.RemoveClientNoLock(loc)
}

func (c *Cell) RemoveClients(loc *publicUtils.Location, amount int) {
	c.Lock()
	defer c.Unlock()
	c.numClients -= amount
	id := loc.GetId()
	value := c.clientLocations[id]
	value.Count -= amount
	if value.Count == 0 {
		delete(c.clientLocations, id)
	}
}

func (c *Cell) RemoveClientsNoLock(loc *publicUtils.Location, amount int) {
	c.numClients -= amount
	id := loc.GetId()
	value := c.clientLocations[id]
	value.Count -= amount
	if value.Count == 0 {
		delete(c.clientLocations, id)
	}
}

func (c *Cell) RemoveClientNoLock(loc *publicUtils.Location) {
	c.numClients--
	id := loc.GetId()
	value := c.clientLocations[id]
	value.Count--
	if value.Count == 0 {
		delete(c.clientLocations, id)
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

func (c *Cell) IterateLocationsNoLock(f func(locId string, loc *ClientLocation) bool) {
	for locId, loc := range c.clientLocations {
		if !f(locId, loc) {
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
