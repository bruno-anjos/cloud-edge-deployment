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

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/environment"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	client "github.com/nm-morais/demmon-client/pkg"
	"github.com/nm-morais/demmon-common/body_types"
	log "github.com/sirupsen/logrus"
)

const (
	connectTimeout = 5 * time.Second
)

var demmonCli *client.DemmonClient

func InitAlternatives() {
	demmonCliConf := client.DemmonClientConf{
		DemmonPort:     environment.DaemonPort,
		DemmonHostAddr: myself.Addr,
		RequestTimeout: environment.ClientRequestTimeout,
	}

	demmonCli = client.New(demmonCliConf)
	err, errChan := demmonCli.ConnectTimeout(connectTimeout)
	if err != nil {
		log.Panic(err)
	}

	go internalUtils.PanicOnErrFromChan(errChan)
}

func setAlternativesHandler(_ http.ResponseWriter, r *http.Request) {
	deployerID := internalUtils.ExtractPathVar(r, nodeIDPathVar)

	reqBody := api.AlternativesRequestBody{}

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		log.Panic(err)
	}

	nodeAlternativesLock.Lock()
	defer nodeAlternativesLock.Unlock()

	nodeAlternatives[deployerID] = reqBody
}

func updateAlternatives() {
	res, err, _, updateChan := demmonCli.SubscribeNodeUpdates()
	if err != nil {
		log.Panic(err)
	}

	go getAlternativesPeriodically(updateChan)

	addNodes(res.Children...)
	addNodes(res.Siblings...)
	if res.Parent != nil {
		addNodes(res.Parent)
	}
}

func addNodes(peers ...*body_types.Peer) {
	for _, peer := range peers {
		addr := peer.IP.String() + ":" + strconv.Itoa(autonomic.Port)

		id, status := autonomicClient.GetID(addr)
		if status != http.StatusOK {
			log.Errorf("got status %d while getting location for %s", status, addr)
		}

		onNodeUp(id, peer.IP.String())
	}
}

func getAlternativesPeriodically(updateChan <-chan body_types.NodeUpdates) {
	for nodeUpdate := range updateChan {
		switch nodeUpdate.Type {
		case body_types.NodeUp:
			addr := nodeUpdate.Peer.IP.String() + ":" + strconv.Itoa(autonomic.Port)

			id, status := autonomicClient.GetID(addr)
			if status != http.StatusOK {
				log.Errorf("got status %d while getting id for %s", status, addr)
			}

			log.Debugf("Alternative Up: %s -> %s", id, nodeUpdate.Peer.IP.String())
			onNodeUp(id, nodeUpdate.Peer.IP.String())
		case body_types.NodeDown:
			log.Debugf("Alternative Down: -> %s", nodeUpdate.Peer.IP.String())
			onNodeDown(nodeUpdate.Peer.IP.String())
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
	depClient := deplFactory.New()
	addr := neighbor.Addr + ":" + strconv.Itoa(deployer.Port)

	status := depClient.SendAlternatives(addr, myself.Addr, alternatives)
	if status != http.StatusOK {
		log.Errorf("got status %d while sending alternatives to %s", status, neighbor.Addr)
	}
}
