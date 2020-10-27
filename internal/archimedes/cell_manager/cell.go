package cell_manager

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
		children        *CellsCollection
		setChildren     *sync.Once
		sync.RWMutex
	}
)

func NewCell(numClients int, clientLocations map[string]*ClientLocation) *Cell {
	return &Cell{
		numClients:      numClients,
		clientLocations: clientLocations,
		children:        nil,
		setChildren:     &sync.Once{},
	}
}

func (c *Cell) AddClient(loc *publicUtils.Location) {
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
	c.numClients--
	id := loc.GetId()
	value := c.clientLocations[id]
	value.Count--
	if value.Count == 0 {
		delete(c.clientLocations, id)
	}
}

func (c *Cell) AddChild(cellId s2.CellID, cell *Cell) {
	c.children.AddCell(cellId, cell)
}

func (c *Cell) HasChildren() bool {
	return c.children != nil
}

func (c *Cell) GetChildren() *CellsCollection {
	return c.children
}

func (c *Cell) GetNumClients() int {
	return c.numClients
}

func (c *Cell) IterateLocations(f func(locId string, loc *ClientLocation) bool) {
	for locId, loc := range c.clientLocations {
		if !f(locId, loc) {
			break
		}
	}
}

func (c *Cell) Clear() {
	c.numClients = 0
	c.clientLocations = nil
	c.children = nil
	c.setChildren = &sync.Once{}
}

func (c *Cell) LockAndLockChildren() {
	c.Lock()
	c.children.Lock()
}

func (c *Cell) UnlockChildrenAndUnlock() {
	c.children.Unlock()
	c.Unlock()
}
