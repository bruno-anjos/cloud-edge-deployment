package deployer

import (
	"net/http"
	"strconv"
	"time"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"

	log "github.com/sirupsen/logrus"
)

var (
	childrenClient = client.NewDeployerClient("")
)

func sendHeartbeatsPeriodically() {
	ticker := time.NewTicker(heartbeatTimeout * time.Second)

	var childrenToRemove []string
	for {
		childrenToRemove = []string{}
		children.Range(func(key, value interface{}) bool {
			childId := key.(string)
			log.Debugf("sending heartbeat to %s", childId)
			child := value.(typeChildrenMapValue)
			childrenClient.SetHostPort(child.Addr + ":" + strconv.Itoa(utils.DeployerPort))
			status := childrenClient.SetParentAlive(myself.Id)
			if status != http.StatusOK {
				log.Errorf("got status %d while telling %s that i was alive", status, child.Id)
				childrenToRemove = append(childrenToRemove, childId)
			}

			return true
		})

		for _, deploymentId := range hTable.getDeployments() {
			depChildren := hTable.getChildren(deploymentId)
			for _, childId := range childrenToRemove {
				if _, ok := depChildren[childId]; ok {
					hTable.removeChild(deploymentId, childId)
					children.Delete(childId)
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
			pTable.removeParent(deadParent.Id)
			renegotiateParent(deadParent, getParentAlternatives(deadParent.Id))
		}
	}
}
