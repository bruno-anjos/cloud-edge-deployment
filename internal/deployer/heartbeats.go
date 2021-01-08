package deployer

import (
	"net/http"
	"strconv"
	"time"

	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"

	log "github.com/sirupsen/logrus"
)

func sendHeartbeatsPeriodically() {
	ticker := time.NewTicker(heartbeatTimeout * time.Second)

	childrenClient := deplFactory.New()

	var childrenToRemove []string

	for {
		childrenToRemove = []string{}

		children.Range(func(key, value interface{}) bool {
			childID := key.(string)
			log.Debugf("sending heartbeat to %s", childID)
			child := value.(typeChildrenMapValue)
			addr := child.Addr + ":" + strconv.Itoa(deployer.Port)

			status := childrenClient.SetParentAlive(addr, myself.ID)
			if status != http.StatusOK {
				log.Errorf("got status %d while telling %s that i was alive", status, child.ID)
				childrenToRemove = append(childrenToRemove, childID)
			}

			return true
		})

		for _, deploymentID := range hTable.getDeployments() {
			depChildren := hTable.getChildren(deploymentID)

			for _, childID := range childrenToRemove {
				if _, ok := depChildren[childID]; ok {
					hTable.removeChild(deploymentID, childID)
					children.Delete(childID)
				}
			}
		}

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
			pTable.removeParent(deadParent.ID)
			renegotiateParent(deadParent, getParentAlternatives(deadParent.ID))
		}
	}
}
