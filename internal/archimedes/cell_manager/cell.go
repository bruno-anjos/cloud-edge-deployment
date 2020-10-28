package cell_manager

import (
	"sync"

	publicUtils "github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
)

type (
	ClientLocation struct {
		Location *publicUtils.Location
		Count    int
	}

	Cell struct {
		numClients      int
		clientLocations map[string]*ClientLocation
		Children        *CellsCollection
		setChildren     *sync.Once
		sync.RWMutex
	}
)

func NewCell(numClients int, clientLocations map[string]*ClientLocation) *Cell {
	return &Cell{
		numClients:      numClients,
		clientLocations: clientLocations,
		Children:        nil,
		setChildren:     &sync.Once{},
	}
}

func (c *Cell) AddClient(loc *publicUtils.Location) {
	c.Lock()
	defer c.Unlock()
	c.AddClientNoLock(loc)
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

func (c *Cell) RemoveClient(loc *publicUtils.Location) {
	c.Lock()
	defer c.Unlock()
	c.RemoveClientNoLock(loc)
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
	c.Children = nil
	c.setChildren = &sync.Once{}
}
