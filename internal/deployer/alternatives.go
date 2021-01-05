package deployer

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	internalUtils "github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"

	log "github.com/sirupsen/logrus"
)

const (
	nodeIPsFilepath = "/node_ips.json"
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

	filePtr, err := os.Open(nodeIPsFilepath)
	if err != nil {
		panic(err)
	}

	var nodeIPs map[string]string

	err = json.NewDecoder(filePtr).Decode(&nodeIPs)
	if err != nil {
		panic(err)
	}

	for {
		<-ticker.C

		vicinity, status := autonomicClient.GetVicinity()
		if status != http.StatusOK {
			continue
		}

		for _, neighbor := range vicinity.Nodes {
			log.Debugf("Alternative: %s -> %s", neighbor.ID, nodeIPs[neighbor.ID])
			onNodeUp(neighbor.ID, nodeIPs[neighbor.ID])
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
