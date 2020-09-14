package deployer

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	log "github.com/sirupsen/logrus"
)

func setAlternativesHandler(_ http.ResponseWriter, r *http.Request) {
	deployerId := utils.ExtractPathVar(r, nodeIdPathVar)

	reqBody := new(api.AlternativesRequestBody)
	err := json.NewDecoder(r.Body).Decode(reqBody)
	if err != nil {
		panic(err)
	}

	nodeAlternativesLock.Lock()
	defer nodeAlternativesLock.Unlock()

	nodeAlternatives[deployerId] = *reqBody
}

func simulateAlternatives() {
	go writeMyselfToAlternatives()
	go loadAlternativesPeriodically()
}

func writeMyselfToAlternatives() {
	ticker := time.NewTicker(30 * time.Second)
	filename := alternativesDir + addPortToAddr(hostname)

	for {
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			_, err = os.Create(filename)
			if err != nil {
				log.Error(err)
			}
		}

		<-ticker.C
	}
}

func loadAlternativesPeriodically() {
	ticker := time.NewTicker(30 * time.Second)

	for {
		<-ticker.C

		files, err := ioutil.ReadDir(alternativesDir)
		if err != nil {
			log.Error(err)
			continue
		}

		for _, f := range files {
			addr := f.Name()
			if addr == hostname {
				continue
			}

			onNodeUp(addr)
		}
	}
}

func sendAlternativesPeriodically() {
	for {
		<-timer.C
		sendAlternatives()
		timer.Reset(sendAlternativesTimeout * time.Second)
	}
}

func sendAlternatives() {
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
	depClient := deployer.NewDeployerClient(neighbor.Addr + ":" + strconv.Itoa(deployer.Port))
	status := depClient.SendAlternatives(myself.Id, alternatives)
	if status != http.StatusOK {
		log.Errorf("got status %d while sending alternatives to %s", status, neighbor.Addr)
	}
}
