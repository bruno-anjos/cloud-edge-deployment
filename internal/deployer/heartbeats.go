package deployer

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	log "github.com/sirupsen/logrus"
)

func sendHeartbeatsPeriodically() {
	ticker := time.NewTicker(heartbeatTimeout * time.Second)

	for {
		children.Range(func(key, value interface{}) bool {
			childId := key.(string)
			log.Debugf("sending heartbeat to %s", childId)
			child := value.(typeChildrenMapValue)
			childrenClient.SetHostPort(child.Addr + ":" + strconv.Itoa(deployer.Port))
			status := childrenClient.SetParentAlive(myself.Id)
			if status != http.StatusOK {
				log.Errorf("got status %d while telling %s that i was alive", status, child.Id)
			}

			return true
		})

		<-ticker.C
	}
}

func checkParentHeartbeatsPeriodically() {
	ticker := time.NewTicker(checkParentsTimeout * time.Second)
	for {
		<-ticker.C
		deadParents := pTable.checkDeadParents()
		if len(deadParents) == 0 {
			log.Debugf("all parents alive")
			continue
		}

		for _, deadParent := range deadParents {
			log.Debugf("dead parent: %+v", deadParent)
			pTable.removeParent(deadParent.Id)
			filename := alternativesDir + deadParent.Addr
			if _, err := os.Stat(filename); os.IsNotExist(err) {
				err = os.Remove(filename)
				log.Error(err)
			}
			renegotiateParent(deadParent)
		}
	}
}
