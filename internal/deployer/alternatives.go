package deployer

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	internalUtils "github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"

	log "github.com/sirupsen/logrus"
)

func setAlternativesHandler(_ http.ResponseWriter, r *http.Request) {
	deployerID := internalUtils.ExtractPathVar(r, nodeIDPathVar)

	reqBody := api.AlternativesRequestBody{}

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		panic(err)
	}

	nodeAlternativesLock.Lock()
	defer nodeAlternativesLock.Unlock()

	nodeAlternatives[deployerID] = reqBody
}

func simulateAlternatives() {
	go loadAlternativesPeriodically()
}

func loadAlternativesPeriodically() {
	const timeout = 30 * time.Second
	ticker := time.NewTicker(timeout)

	for {
		<-ticker.C

		vicinity, status := autonomicClient.GetVicinity()
		if status != http.StatusOK {
			continue
		}

		for _, neighbor := range vicinity.Nodes {
			onNodeUp(neighbor.ID, neighbor.Addr)
		}
	}
}

func sendAlternativesPeriodically() {
	for {
		<-timer.C
		sendAlternatives()

		if !timer.Stop() {
			<-timer.C
		}

		timer.Reset(sendAlternativesTimeout)
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
	depClient := deplFactory.New(neighbor.Addr + ":" + strconv.Itoa(deployer.Port))

	status := depClient.SendAlternatives(myself.Addr, alternatives)
	if status != http.StatusOK {
		log.Errorf("got status %d while sending alternatives to %s", status, neighbor.Addr)
	}
}
