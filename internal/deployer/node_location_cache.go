package deployer

import (
	"sync"

	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/golang/geo/s2"
)

type (
	typeNodeLocationCache = s2.CellID

	nodeLocationCache struct {
		sync.Map
	}
)

func (nc *nodeLocationCache) get(node *utils.Node) (location s2.CellID) {
	value, ok := nc.Load(node.ID)
	if !ok {
		nc.Store(node.ID, location)
	} else {
		location = value.(typeNodeLocationCache)
	}

	return
}
