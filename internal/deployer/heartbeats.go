package deployer

import (
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

func sendHeartbeatsPeriodically() {
	ticker := time.NewTicker(heartbeatTimeout * time.Second)

	for {
		children.Range(func(key, value interface{}) bool {
			child := value.(typeChildrenMapValue)
			childrenClient.SetHostPort(child.Addr, Port)
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
		deadParents := parentsTable.CheckDeadParents()
		if len(deadParents) == 0 {
			log.Debugf("all parents alive")
			continue
		}

		for _, deadParent := range deadParents {
			log.Debugf("dead parent: %+v", deadParent)
			parentsTable.RemoveParent(deadParent.Id)
			filename := alternativesDir + deadParent.Addr
			if _, err := os.Stat(filename); os.IsNotExist(err) {
				os.Remove(filename)
			}
			renegotiateParent(deadParent)
		}
	}
}
