package deployer

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/scheduler"
	"github.com/docker/go-connections/nat"
	"github.com/golang/geo/s2"
	log "github.com/sirupsen/logrus"

	"gopkg.in/yaml.v3"
)

type (
	typeMyAlternativesMapValue = *utils.Node
	typeChildrenMapValue       = *utils.Node

	typeSuspectedChildMapKey = string
)

// Timeouts
const (
	sendAlternativesTimeout = 30
	checkParentsTimeout     = 30
	heartbeatTimeout        = 10
	extendAttemptTimeout    = 5
	waitForNewParentTimeout = 60
)

const (
	fallbackFilename = "fallback.txt"
	maxHopsToLookFor = 5

	maxHopslocationHorizon = 3
)

var (
	autonomicClient  = autonomic.NewAutonomicClient(autonomic.DefaultHostPort)
	archimedesClient = archimedes.NewArchimedesClient(archimedes.DefaultHostPort)
	schedulerClient  = scheduler.NewSchedulerClient(scheduler.DefaultHostPort)
)

var (
	hostname string
	location s2.Cell
	fallback string
	myself   *utils.Node

	myAlternatives       sync.Map
	nodeAlternatives     map[string][]*utils.Node
	nodeAlternativesLock sync.RWMutex

	hTable *hierarchyTable
	pTable *parentsTable

	suspectedChild       sync.Map
	suspectedDeployments sync.Map
	children             sync.Map

	nodeLocCache *nodeLocationCache

	timer *time.Timer
)

func init() {
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		panic(err)
	}

	myself = utils.NewNode(hostname, hostname)

	myAlternatives = sync.Map{}
	nodeAlternatives = map[string][]*utils.Node{}
	nodeAlternativesLock = sync.RWMutex{}
	hTable = newHierarchyTable()
	pTable = newParentsTable()

	suspectedChild = sync.Map{}
	suspectedDeployments = sync.Map{}
	children = sync.Map{}

	nodeLocCache = &nodeLocationCache{}

	timer = time.NewTimer(sendAlternativesTimeout * time.Second)

	// TODO change this for location from lower API
	fallback = loadFallbackHostname(fallbackFilename)
	log.Debugf("loaded fallback %s", fallback)

	simulateAlternatives()

	var (
		locationId s2.CellID
		status     int
	)
	for status != http.StatusOK {
		locationId, status = autonomicClient.GetLocation()
		time.Sleep(10 * time.Second)
	}

	location = s2.CellFromCellID(locationId)

	log.Debugf("got location %f", location)

	go sendHeartbeatsPeriodically()
	go sendAlternativesPeriodically()
	go checkParentHeartbeatsPeriodically()
}

func propagateLocationToHorizonHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)

	var reqBody api.PropagateLocationToHorizonRequestBody
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		panic(err)
	}

	log.Debugf("got location from %s for deployment %s (%s)", reqBody.ChildId, deploymentId, reqBody.Operation)

	switch reqBody.Operation {
	case api.Add:
		archimedesClient.AddDeploymentNode(deploymentId, reqBody.ChildId, reqBody.Location, false)
	case api.Remove:
		archimedesClient.DeleteDeploymentNode(deploymentId, reqBody.ChildId)
	}

	parent := hTable.getParent(deploymentId)
	if reqBody.TTL+1 >= maxHopslocationHorizon || parent == nil {
		return
	}

	deplClient := deployer.NewDeployerClient(parent.Id + ":" + strconv.Itoa(deployer.Port))
	log.Debugf("propagating %s location for deployments %+v to %s", reqBody.ChildId, deploymentId, parent.Id)
	deplClient.PropagateLocationToHorizon(deploymentId, reqBody.ChildId, reqBody.Location, reqBody.TTL+1,
		reqBody.Operation)
}

func extendDeploymentToHandler(w http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)
	targetAddr := utils.ExtractPathVar(r, nodeIdPathVar)

	log.Debugf("handling request to extend deployment %s to %s", deploymentId, targetAddr)

	if !hTable.hasDeployment(deploymentId) {
		log.Debugf("deployment %s does not exist, ignoring extension request", deploymentId)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var reqBody api.ExtendDeploymentRequestBody
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		panic(err)
	}

	go attemptToExtend(deploymentId, targetAddr, reqBody.Config, 0, nil, reqBody.ExploringTTL)
}

func childDeletedDeploymentHandler(_ http.ResponseWriter, r *http.Request) {
	log.Debugf("handling child deleted request")
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)
	childId := utils.ExtractPathVar(r, nodeIdPathVar)

	hTable.removeChild(deploymentId, childId)
	children.Delete(childId)

	autoClient := autonomic.NewAutonomicClient("localhost:" + strconv.Itoa(autonomic.Port))
	autoClient.BlacklistNodes(deploymentId, myself.Id, childId)
}

func getDeploymentsHandler(w http.ResponseWriter, _ *http.Request) {
	deployments := hTable.getDeployments()
	utils.SendJSONReplyOK(w, deployments)
}

func registerDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling register deployment request")

	var registerBody api.RegisterDeploymentRequestBody
	err := json.NewDecoder(r.Body).Decode(&registerBody)
	if err != nil {
		panic(err)
	}

	deploymentDTO := registerBody.DeploymentConfig
	deploymentId := deploymentDTO.DeploymentId

	var deploymentYAML api.DeploymentYAML
	err = yaml.Unmarshal(deploymentDTO.DeploymentYAMLBytes, &deploymentYAML)
	if err != nil {
		panic(err)
	}

	parent := hTable.getParent(deploymentId)

	parentId := "nil"
	if parent != nil {
		parentId = parent.Id
	}

	deploymentParentId := "nil"
	if deploymentDTO.Parent != nil {
		deploymentParentId = deploymentDTO.Parent.Id
	}

	log.Debugf("my parent is %s and the presented parent is %s", parentId,
		deploymentParentId)

	if hTable.hasDeployment(deploymentId) {
		parentsMatch := parent != nil && deploymentDTO.Parent != nil && parentId == deploymentParentId
		parentDead := parent != nil && !pTable.hasParent(parentId)

		log.Debugf("conditions: %t, %t", parentsMatch, parentDead)

		if !parentsMatch && !parentDead {
			shouldTakeChildren := len(deploymentDTO.Children) > 0 && checkChildren(parent, deploymentDTO.Children...)
			if !shouldTakeChildren {
				// case where i have the deployment, its not my parent speaking to me, my parent is not dead
				// and i should not take the children
				w.WriteHeader(http.StatusConflict)
				return
			}
			log.Debug("won't take deployment but should take children")
		}
	}

	hTable.Lock()

	parent = hTable.getParent(deploymentId)
	parentId = "nil"
	if parent != nil {
		parentId = parent.Id
	}

	log.Debugf("my parent is %s and the presented parent is %s", parentId,
		deploymentParentId)

	alreadyHadDeployment := false
	if hTable.hasDeployment(deploymentDTO.DeploymentId) {
		alreadyHadDeployment = true
		parentsMatch := parent != nil && deploymentDTO.Parent != nil && parentId == deploymentParentId
		parentDead := parent != nil && !pTable.hasParent(parentId)

		if !parentsMatch && !parentDead {
			shouldTakeChildren := len(deploymentDTO.Children) > 0 && checkChildren(parent, deploymentDTO.Children...)

			if !shouldTakeChildren {
				// after locking to add guarantee that in the meanwhile it wasn't added
				w.WriteHeader(http.StatusConflict)
				hTable.Unlock()
				return
			}

			log.Debugf("will take children %+v for deployment %s", deploymentDTO.Children, deploymentId)
			takeChildren(deploymentId, parent, deploymentDTO.Children...)

			w.WriteHeader(http.StatusNoContent)
			hTable.Unlock()
			return
		}
	}

	canTake := checkChildren(parent, deploymentDTO.Children...)
	if !canTake {
		w.WriteHeader(http.StatusBadRequest)
		hTable.Unlock()
		return
	}

	if alreadyHadDeployment {
		hTable.updateDeployment(deploymentId, parent, deploymentDTO.Grandparent)
	} else {
		success := hTable.addDeployment(deploymentDTO, registerBody.ExploringTTL)
		if !success {
			log.Debugf("failed adding deployment %s", deploymentId)
			w.WriteHeader(http.StatusConflict)
			hTable.Unlock()
			return
		}
	}

	if deploymentDTO.Parent != nil {
		if !pTable.hasParent(deploymentDTO.Parent.Id) {
			pTable.addParent(deploymentDTO.Parent)
		}
	}

	hTable.Unlock()

	if !alreadyHadDeployment {
		deployment := deploymentYAMLToDeployment(&deploymentYAML, deploymentDTO.Static)
		go addDeploymentAsync(deployment, deploymentDTO.DeploymentId)
	}

	takeChildren(deploymentId, parent, deploymentDTO.Children...)
}

func takeChildren(deploymentId string, parent *utils.Node, children ...*utils.Node) {
	deplClient := deployer.NewDeployerClient("")

	for _, child := range children {
		deplClient.SetHostPort(child.Id + ":" + strconv.Itoa(deployer.Port))
		status := deplClient.WarnThatIAmParent(deploymentId, myself, parent)
		if status == http.StatusConflict {
			log.Debugf("can not be %s parent since it already has a live parent", child.Id)
			continue
		} else if status != http.StatusOK {
			log.Errorf("got status code %d while telling %s that im his parent for %s", status, child.Id,
				deploymentId)
			continue
		}
		hTable.addChild(deploymentId, child, false)
	}
}

func checkChildren(parent *utils.Node, children ...*utils.Node) bool {
	for _, child := range children {
		if parent != nil && child.Id != parent.Id {
			log.Debugf("can take child %s, my parent is %s", child.Id, parent.Id)
		} else {
			// if any of the children is my parent i can't take them
			log.Debugf("rejecting child %s", child)
			return false
		}
	}

	return true
}

func deleteDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)

	log.Debugf("deleting deployment %s", deploymentId)

	parent := hTable.getParent(deploymentId)
	if parent != nil {
		client := deployer.NewDeployerClient(parent.Addr + ":" + strconv.Itoa(deployer.Port))
		status := client.ChildDeletedDeployment(deploymentId, myself.Id)
		if status != http.StatusOK {
			log.Errorf("got status %d from child deleted deployment", status)
			w.WriteHeader(status)
			return
		}
		pTable.decreaseParentCount(parent.Id)
	}

	hTable.removeDeployment(deploymentId)

	go deleteDeploymentAsync(deploymentId)
}

func addNodeHandler(_ http.ResponseWriter, r *http.Request) {
	var nodeAddr string
	err := json.NewDecoder(r.Body).Decode(&nodeAddr)
	if err != nil {
		panic(err)
	}

	onNodeUp(nodeAddr)
}

func whoAreYouHandler(w http.ResponseWriter, _ *http.Request) {
	utils.SendJSONReplyOK(w, myself.Id)
}

func getFallbackHandler(w http.ResponseWriter, _ *http.Request) {
	var respBody api.GetFallbackResponseBody
	respBody = fallback

	utils.SendJSONReplyOK(w, respBody)
}

func hasDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)
	if !hTable.hasDeployment(deploymentId) {
		w.WriteHeader(http.StatusNotFound)
	}
}

// TODO function simulating lower API
func getNodeCloserTo(locations []s2.CellID, maxHopsToLookFor int,
	excludeNodes map[string]interface{}) (closest string, found bool) {
	closest = autonomicClient.GetClosestNode(locations, excludeNodes)
	found = closest != ""
	return
}

func addDeploymentAsync(deployment *Deployment, deploymentId string) {
	log.Debugf("adding deployment %s", deploymentId)

	status := archimedesClient.RegisterDeployment(deploymentId, deployment.Ports)
	if status != http.StatusOK {
		log.Errorf("got status code %d from archimedes", status)
		return
	}

	for i := 0; i < deployment.NumberOfInstances; i++ {
		status = schedulerClient.StartInstance(deploymentId, deployment.Image, deployment.Ports, deployment.Static,
			deployment.EnvVars, deployment.Command)
		if status != http.StatusOK {
			log.Errorf("got status code %d from scheduler", status)

			status = archimedesClient.DeleteDeployment(deploymentId)
			if status != http.StatusOK {
				log.Error("error deleting deployment that failed initializing")
			}

			hTable.removeDeployment(deploymentId)
			return
		}
	}
}

func deleteDeploymentAsync(deploymentId string) {
	autonomicClient.DeleteDeployment(deploymentId)
}

func deploymentYAMLToDeployment(deploymentYAML *api.DeploymentYAML, static bool) *Deployment {
	log.Debugf("%+v", deploymentYAML)

	numContainers := len(deploymentYAML.Containers)
	if numContainers > 1 {
		panic("more than one container per deployment is not supported")
	} else if numContainers == 0 {
		panic("no container provided")
	}

	containerSpec := deploymentYAML.Containers[0]

	envVars := make([]string, len(containerSpec.Env))
	for i, envVar := range containerSpec.Env {
		envVars[i] = envVar.Name + "=" + envVar.Value
	}

	ports := nat.PortSet{}
	for _, port := range containerSpec.Ports {
		natPort, err := nat.NewPort(utils.TCP, port.ContainerPort)
		if err != nil {
			panic(err)
		}

		ports[natPort] = struct{}{}
	}

	deployment := Deployment{
		DeploymentId:      deploymentYAML.DeploymentName,
		NumberOfInstances: deploymentYAML.Replicas,
		Image:             containerSpec.Image,
		Command:           containerSpec.Command,
		EnvVars:           envVars,
		Ports:             ports,
		Static:            static,
		Lock:              &sync.RWMutex{},
	}

	log.Debugf("%+v", deployment)

	return &deployment
}

func addNode(nodeDeployerId, addr string) bool {
	if nodeDeployerId == "" {
		panic("error while adding node up")
	}

	if nodeDeployerId == myself.Id {
		return true
	}

	suspectedChild.Delete(nodeDeployerId)

	_, ok := myAlternatives.Load(nodeDeployerId)
	if ok {
		return true
	}

	log.Debugf("added node %s", nodeDeployerId)

	neighbor := utils.NewNode(nodeDeployerId, addr)

	myAlternatives.Store(nodeDeployerId, neighbor)
	return true
}

// TODO function simulation lower API
// Node up is only triggered for nodes that appeared one hop away
func onNodeUp(addr string) {
	var (
		id  string
		err error
	)
	if strings.Contains(addr, ":") {
		id, _, err = net.SplitHostPort(addr)
		if err != nil {
			panic(err)
		}
	} else {
		id = addr
	}
	addNode(id, id)
	sendAlternatives()
	if !timer.Stop() {
		<-timer.C
	}
	timer.Reset(sendAlternativesTimeout * time.Second)
}

func getParentAlternatives(parentId string) (alternatives map[string]*utils.Node) {
	nodeAlternativesLock.RLock()
	defer nodeAlternativesLock.RUnlock()

	alternatives = map[string]*utils.Node{}

	for _, alternative := range nodeAlternatives[parentId] {
		alternatives[alternative.Id] = alternative
	}

	return
}
