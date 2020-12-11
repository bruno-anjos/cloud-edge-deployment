package deployer

import (
	"net/http"
	"strconv"
	"sync"

	"github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/golang/geo/s2"
	log "github.com/sirupsen/logrus"
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
		autoClient := autoFactory.New(node.Addr + ":" + strconv.Itoa(autonomic.Port))

		var status int

		location, status = autoClient.GetLocation()
		if status != http.StatusOK {
			log.Errorf("got %d while trying to get %s location", status, node.ID)

			return 0
		}

		nc.Store(node.ID, location)
	} else {
		location = value.(typeNodeLocationCache)
	}

	return
}
