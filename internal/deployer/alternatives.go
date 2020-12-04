package deployer

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"

	log "github.com/sirupsen/logrus"
)

func setAlternativesHandler(_ http.ResponseWriter, r *http.Request) {
	deployerId := utils.ExtractPathVar(r, nodeIdPathVar)

	reqBody := api.AlternativesRequestBody{}
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		panic(err)
	}

	nodeAlternativesLock.Lock()
	defer nodeAlternativesLock.Unlock()

	nodeAlternatives[deployerId] = reqBody
}

func simulateAlternatives() {
	go loadAlternativesPeriodically()
}

func loadAlternativesPeriodically() {
	ticker := time.NewTicker(30 * time.Second)

	for {
		<-ticker.C

		vicinity, status := autonomicClient.GetVicinity()
		if status != http.StatusOK {
			continue
		}

		for _, neighbor := range vicinity.Nodes {
			onNodeUp(neighbor.Id, neighbor.Addr)
		}
	}
}

func sendAlternativesPeriodically() {
	for {
		// TODO not perfect
		<-timer.C
		sendAlternatives()
		if !timer.Stop() {
			<-timer.C
		}
		timer.Reset(sendAlternativesTimeout * time.Second)
	}
}

func sendAlternatives() {
	log.Debug("sending alternatives")
	var alternatives []*utils.Node
	myAlternatives.Range(func(key, value interface{}) bool {
		neighbor := value.(typeMyAlternativesMapValue)
		alternatives = append(alternatives, neighbor)
		return true
	})

	children.Range(func(key, value interface{}) bool {
		neighbor := value.(typeChildrenMapValue)
		sendAlternativesTo(neighbor, alternatives)
		return true
	})
}

func sendAlternativesTo(neighbor *utils.Node, alternatives []*utils.Node) {
	depClient :=  deplFactory.New(neighbor.Addr + ":" + strconv.Itoa(utils.DeployerPort))
	status := depClient.SendAlternatives(myself.Addr, alternatives)
	if status != http.StatusOK {
		log.Errorf("got status %d while sending alternatives to %s", status, neighbor.Addr)
	}
}
