package deployer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/servers"
	internalUtils "github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/scheduler"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"

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

// Timeouts.
const (
	sendAlternativesTimeout = 30 * time.Second
	checkParentsTimeout     = 30
	heartbeatTimeout        = 10
	extendAttemptTimeout    = 5
	waitForNewParentTimeout = 60
)

const (
	fallbackFilename = "fallback.json"
	maxHopsToLookFor = 5

	maxHopslocationHorizon = 3
)

const (
	deploymentNameFmt = "DEPLOYMENT_NAME"
	replicaNumFmt     = "REPLICA_NUM"
	nilString         = "nil"
)

var (
	autonomicClient  autonomic.Client
	archimedesClient archimedes.Client
	schedulerClient  scheduler.Client
)

var (
	location s2.Cell
	fallback *utils.Node
	myself   *utils.Node

	myAlternatives       sync.Map
	nodeAlternatives     map[string][]*utils.Node
	nodeAlternativesLock sync.RWMutex

	hTable *hierarchyTable
	pTable *parentsTable

	suspectedChild       sync.Map
	suspectedDeployments sync.Map
	children             sync.Map

	nodeLocCache sync.Map

	timer *time.Timer

	autoFactory  autonomic.ClientFactory
	archFactory  archimedes.ClientFactory
	deplFactory  deployer.ClientFactory
	schedFactory scheduler.ClientFactory

	nodeIP     string
	ready      chan interface{}
	closeReady sync.Once

	startRecording sync.Once
)

func InitServer(autoFactoryAux autonomic.ClientFactory, archFactoryAux archimedes.ClientFactory,
	deplFactoryAux deployer.ClientFactory, schedFactoryAux scheduler.ClientFactory) {
	go instanceHeartbeatChecker()

	log.SetLevel(log.DebugLevel)

	myself = utils.NodeFromEnv()

	autoFactory = autoFactoryAux
	archFactory = archFactoryAux
	deplFactory = deplFactoryAux
	schedFactory = schedFactoryAux

	autonomicClient = autoFactory.New()
	archimedesClient = archFactory.New(servers.ArchimedesLocalHostPort)
	schedulerClient = schedFactory.New()

	myAlternatives = sync.Map{}
	nodeAlternatives = map[string][]*utils.Node{}
	nodeAlternativesLock = sync.RWMutex{}
	hTable = newHierarchyTable()
	pTable = newParentsTable()

	timer = time.NewTimer(sendAlternativesTimeout)

	fallback = loadFallbackHostname(fallbackFilename)
	log.Debugf("loaded fallback %+v", fallback)

	ready = make(chan interface{})

	go func() {
		closeReady = sync.Once{}
		<-ready
		updateAlternatives()
	}()

	locationToken, ok := os.LookupEnv(utils.LocationEnvVarName)
	if !ok {
		log.Panic("no location env var set")
	}

	location = s2.CellFromCellID(s2.CellIDFromToken(locationToken))

	log.Debugf("got location %s", location.ID().ToToken())

	var exists bool
	nodeIP, exists = os.LookupEnv(utils.NodeIPEnvVarName)

	if !exists {
		log.Panic("no IP env var")
	}

	InitAlternatives()

	go sendHeartbeatsPeriodically()
	go sendAlternativesPeriodically()
	go checkParentHeartbeatsPeriodically()
}

func processTimeString(timeString string) time.Duration {
	timeValue, err := strconv.Atoi(timeString[0 : len(timeString)-1])
	if err != nil {
		log.Panic(err)
	}

	timeSuffix := timeString[len(timeString)-1]

	var duration time.Duration

	switch timeSuffix {
	case 's', 'S':
		duration = time.Second
	case 'm', 'M':
		duration = time.Minute
	case 'h', 'H':
		duration = time.Hour
	default:
		log.Panicf("invalid time suffix %c", timeSuffix)
	}

	return time.Duration(timeValue) * duration
}

func record(totalDurationString, timeoutString string) {
	totalDuration := processTimeString(totalDurationString)
	timeout := processTimeString(timeoutString)

	numCycles := int(totalDuration / timeout)

	err := os.Mkdir(fmt.Sprintf("%s/%s", tableDirPath, myself.ID), os.ModePerm)
	if err != nil {
		log.Panic(err)
	}

	for i := 0; i < numCycles; i++ {
		recordHierarchyTable(i + 1)
		time.Sleep(timeout)
	}
}

const tableDirPath = "/tables"

func recordHierarchyTable(count int) {
	toSave := hTable.toDTO()

	jsonString, err := json.Marshal(toSave)
	if err != nil {
		log.Panic(err)
	}

	filePath := fmt.Sprintf("%s/%s/%d.json", tableDirPath, myself.ID, count)

	err = ioutil.WriteFile(filePath, jsonString, os.ModePerm)
	if err != nil {
		log.Panic(err)
	}
}

func startRecordingHandler(_ http.ResponseWriter, r *http.Request) {
	var reqBody struct {
		Duration string
		Timeout  string
	}

	log.Debugf("Starting recording...")

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		log.Panic(err)
	}

	startRecording.Do(func() {
		go record(reqBody.Duration, reqBody.Timeout)
	})
}

func propagateLocationToHorizonHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentID := internalUtils.ExtractPathVar(r, deploymentIDPathVar)

	var reqBody api.PropagateLocationToHorizonRequestBody

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		panic(err)
	}

	log.Debugf("got location from %s for deployment %s (%s)", reqBody.Child.ID, deploymentID, reqBody.Operation)

	switch reqBody.Operation {
	case api.Add:
		archimedesClient.AddDeploymentNode(servers.ArchimedesLocalHostPort, deploymentID, reqBody.Child,
			reqBody.Location, false)
	case api.Remove:
		archimedesClient.DeleteDeploymentNode(servers.ArchimedesLocalHostPort, deploymentID, reqBody.Child.ID)
	}

	parent := hTable.getParent(deploymentID)
	if reqBody.TTL+1 >= maxHopslocationHorizon || parent == nil {
		return
	}

	addr := parent.Addr + ":" + strconv.Itoa(deployer.Port)
	deplClient := deplFactory.New()
	log.Debugf("propagating %s location for deployments %+v to %s", reqBody.Child.ID, deploymentID, parent.ID)
	deplClient.PropagateLocationToHorizon(addr, deploymentID, reqBody.Child, reqBody.Location, reqBody.TTL+1,
		reqBody.Operation)
}

func extendDeploymentToHandler(w http.ResponseWriter, r *http.Request) {
	deploymentID := internalUtils.ExtractPathVar(r, deploymentIDPathVar)

	var reqBody api.ExtendDeploymentRequestBody

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		panic(err)
	}

	nodeID := ""
	if reqBody.Node != nil {
		nodeID = reqBody.Node.ID
	}

	log.Debugf("handling request to extend deployment %s to %s", deploymentID, nodeID)

	if !hTable.hasDeployment(deploymentID) {
		log.Debugf("deployment %s does not exist, ignoring extension request", deploymentID)
		w.WriteHeader(http.StatusNotFound)

		return
	}

	go attemptToExtend(deploymentID, reqBody.Node, reqBody.Config, 0, nil, reqBody.ExploringTTL)
}

func childDeletedDeploymentHandler(_ http.ResponseWriter, r *http.Request) {
	log.Debugf("handling child deleted request")

	deploymentID := internalUtils.ExtractPathVar(r, deploymentIDPathVar)
	childID := internalUtils.ExtractPathVar(r, nodeIDPathVar)

	hTable.removeChild(deploymentID, childID)
	children.Delete(childID)

	autonomicClient.BlacklistNodes(servers.AutonomicLocalHostPort, deploymentID, myself.ID, []string{childID},
		map[string]struct{}{myself.ID: {}})
}

func getDeploymentsHandler(w http.ResponseWriter, _ *http.Request) {
	deployments := hTable.getDeployments()
	internalUtils.SendJSONReplyOK(w, deployments)
}

func checkIfShouldTakeChildren(myParent *utils.Node,
	deploymentDTO *api.DeploymentDTO) (shouldTakeChildren, conflict bool) {
	parentsMatch := myParent != nil && deploymentDTO.Parent != nil && myParent.ID == deploymentDTO.Parent.ID
	parentDead := myParent != nil && !pTable.hasParent(myParent.ID)

	log.Debugf("conditions: %t, %t", parentsMatch, parentDead)

	if !parentsMatch && !parentDead {
		shouldTakeChildren = len(deploymentDTO.Children) > 0 && checkChildren(myParent, deploymentDTO.Children...)
		if !shouldTakeChildren {
			// case where i have the deployment, its not my parent speaking to me, my parent is not dead
			// and i should not take the children
			conflict = true

			return shouldTakeChildren, conflict
		}

		log.Debug("won't take deployment but should take children")
	}

	return shouldTakeChildren, conflict
}

func getDeploymentYAML(deploymentDTO *api.DeploymentDTO) *api.DeploymentYAML {
	var deploymentYAML api.DeploymentYAML

	err := yaml.Unmarshal(deploymentDTO.DeploymentYAMLBytes, &deploymentYAML)
	if err != nil {
		log.Debug(deploymentDTO.DeploymentYAMLBytes)
		panic(err)
	}

	return &deploymentYAML
}

func logParentReceivedAndDeploymentParent(parent *utils.Node, deploymentDTO *api.DeploymentDTO) {
	parentID := nilString
	if parent != nil {
		parentID = parent.ID
	}

	deploymentParentID := nilString
	if deploymentDTO.Parent != nil {
		deploymentParentID = deploymentDTO.Parent.ID
	}

	log.Debugf("my parent is %s and the presented parent is %s", parentID,
		deploymentParentID)
}

func registerDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling register deployment request")

	var registerBody api.RegisterDeploymentRequestBody

	err := json.NewDecoder(r.Body).Decode(&registerBody)
	if err != nil {
		log.Panic(err)
	}

	deploymentDTO := registerBody.DeploymentConfig
	deploymentID := deploymentDTO.DeploymentID
	deploymentYAML := getDeploymentYAML(deploymentDTO)

	parent := hTable.getParent(deploymentID)
	logParentReceivedAndDeploymentParent(parent, deploymentDTO)

	if hTable.hasDeployment(deploymentID) {
		_, conflict := checkIfShouldTakeChildren(parent, deploymentDTO)
		if conflict {
			w.WriteHeader(http.StatusConflict)

			return
		}
	}

	hTable.Lock()

	parent = hTable.getParent(deploymentID)
	logParentReceivedAndDeploymentParent(parent, deploymentDTO)

	alreadyHadDeployment := false
	if hTable.hasDeployment(deploymentID) {
		alreadyHadDeployment = true

		shouldTakeChildren, conflict := checkIfShouldTakeChildren(parent, deploymentDTO)
		if conflict {
			w.WriteHeader(http.StatusConflict)
			hTable.Unlock()

			return
		}

		if shouldTakeChildren {
			log.Debugf("will take children %+v for deployment %s", deploymentDTO.Children, deploymentID)
			takeChildren(deploymentID, parent, deploymentDTO.Children...)

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
		hTable.updateDeployment(deploymentID, parent, deploymentDTO.Grandparent)
	} else {
		success := hTable.addDeployment(deploymentDTO, deploymentYAML.DepthFactor, registerBody.ExploringTTL)
		if !success {
			log.Debugf("failed adding deployment %s", deploymentID)
			w.WriteHeader(http.StatusConflict)
			hTable.Unlock()

			return
		}
	}

	if deploymentDTO.Parent != nil {
		if !pTable.hasParent(deploymentDTO.Parent.ID) {
			pTable.addParent(deploymentDTO.Parent)
		}
	}

	hTable.Unlock()

	if !alreadyHadDeployment {
		d := deploymentYAMLToDeployment(deploymentYAML, deploymentDTO.Static)
		go addDeploymentAsync(d, deploymentDTO.DeploymentID, deploymentYAML.InstanceNameFmt)
	}

	takeChildren(deploymentID, parent, deploymentDTO.Children...)
}

func takeChildren(deploymentID string, parent *utils.Node, children ...*utils.Node) {
	deplClient := deplFactory.New()

	for _, child := range children {
		addr := child.Addr + ":" + strconv.Itoa(deployer.Port)

		status := deplClient.WarnThatIAmParent(addr, deploymentID, myself, parent)
		if status == http.StatusConflict {
			log.Debugf("can not be %s parent since it already has a live parent", child.ID)

			continue
		} else if status != http.StatusOK {
			log.Errorf("got status code %d while telling %s that im his parent for %s", status, child.ID,
				deploymentID)

			continue
		}

		hTable.addChild(deploymentID, child, false)
	}
}

func checkChildren(parent *utils.Node, children ...*utils.Node) bool {
	for _, child := range children {
		if parent != nil && child.ID != parent.ID {
			log.Debugf("can take child %s, my parent is %s", child.ID, parent.ID)
		} else {
			// if any of the children is my parent i can't take them
			log.Debugf("rejecting child %s", child)

			return false
		}
	}

	return true
}

func deleteDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	deploymentID := internalUtils.ExtractPathVar(r, deploymentIDPathVar)

	log.Debugf("handling request to delete deployment %s", deploymentID)

	success, status := deleteDeployment(deploymentID)
	if !success {
		log.Warnf("failed to delete %s with status %d", deploymentID, status)

		w.WriteHeader(status)
		return
	}
}

func whoAreYouHandler(w http.ResponseWriter, _ *http.Request) {
	internalUtils.SendJSONReplyOK(w, myself.ID)
}

func getFallbackHandler(w http.ResponseWriter, _ *http.Request) {
	log.Debugf("handling get fallback request")

	respBody := *fallback

	log.Debugf("sending %+v", fallback)

	internalUtils.SendJSONReplyOK(w, respBody)
}

func hasDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	deploymentID := internalUtils.ExtractPathVar(r, deploymentIDPathVar)
	if !hTable.hasDeployment(deploymentID) {
		w.WriteHeader(http.StatusNotFound)
	}
}

func setReadyHandler(_ http.ResponseWriter, _ *http.Request) {
	select {
	case <-ready:
	default:
		closeReady.Do(func() {
			close(ready)
		})
	}
}

// Function simulating lower API.
func getNodeCloserTo(locations []s2.CellID, _ int,
	excludeNodes map[string]interface{}) (closest *utils.Node, found bool) {
	closest = autonomicClient.GetClosestNode(servers.AutonomicLocalHostPort, locations, excludeNodes)
	found = closest != nil

	return
}

func fmtInstanceName(instanceFmt []string, deploymentID string, replicaNum int) string {
	var result []string

	for _, fmtString := range instanceFmt {
		if fmtString[0] != '$' {
			result = append(result, fmtString)

			continue
		}

		switch fmtString[1:] {
		case deploymentNameFmt:
			result = append(result, deploymentID)
		case replicaNumFmt:
			result = append(result, strconv.Itoa(replicaNum))
		}
	}

	return strings.Join(result, "")
}

func addDeploymentAsync(deployment *deployment, deploymentID string, instanceNameFmt []string) {
	log.Debugf("adding deployment %s (ni: %d, s: %t, fmt: %+v)", deploymentID, deployment.NumberOfInstances,
		deployment.Static, instanceNameFmt)

	status := archimedesClient.RegisterDeployment(servers.ArchimedesLocalHostPort, deploymentID, deployment.Ports,
		myself)
	if status != http.StatusOK {
		log.Errorf("got status code %d from archimedes", status)

		return
	}

	for i := 0; i < deployment.NumberOfInstances; i++ {
		var instanceName string
		if len(instanceNameFmt) == 0 {
			instanceName = ""
		} else {
			instanceName = fmtInstanceName(instanceNameFmt, deploymentID, i)
			log.Debugf("formatted instance name: %s", instanceName)
		}

		status = schedulerClient.StartInstance(servers.SchedulerLocalHostPort, deploymentID, deployment.Image,
			instanceName, deployment.Ports, i,
			deployment.Static, deployment.EnvVars, deployment.Command)
		if status != http.StatusOK {
			log.Errorf("got status code %d from scheduler", status)

			status = archimedesClient.DeleteDeployment(servers.ArchimedesLocalHostPort, deploymentID)
			if status != http.StatusOK {
				log.Error("error deleting deployment that failed initializing")
			}

			hTable.removeDeployment(deploymentID)

			return
		}
	}
}

func deploymentYAMLToDeployment(deploymentYAML *api.DeploymentYAML, static bool) *deployment {
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
		natPort, err := nat.NewPort(servers.TCP, port.ContainerPort)
		if err != nil {
			panic(err)
		}

		ports[natPort] = struct{}{}
	}

	d := deployment{
		DeploymentID:      deploymentYAML.DeploymentName,
		NumberOfInstances: deploymentYAML.Replicas,
		Image:             containerSpec.Image,
		Command:           containerSpec.Command,
		EnvVars:           envVars,
		Ports:             ports,
		Static:            static,
		Lock:              &sync.RWMutex{},
	}

	log.Debugf("%+v", d)

	return &d
}

func addNode(nodeDeployerID, addr string) (success bool) {
	success = false

	if nodeDeployerID == "" {
		log.Warn("error while adding node up")
		return
	}

	if nodeDeployerID == myself.ID {
		return
	}

	suspectedChild.Delete(nodeDeployerID)

	_, ok := myAlternatives.Load(nodeDeployerID)
	if ok {
		return
	}

	log.Debugf("added node %s", nodeDeployerID)
	success = true

	neighbor := utils.NewNode(nodeDeployerID, addr)

	myAlternatives.Store(nodeDeployerID, neighbor)

	return
}

func removeNode(nodeAddr string) {
	myAlternatives.Range(func(key, value interface{}) bool {
		node := value.(*utils.Node)

		if node.Addr == nodeAddr {
			myAlternatives.Delete(node.ID)
			return false
		}

		return true
	})
}

func onNodeUp(id, addr string) {
	success := addNode(id, addr)
	if !success {
		return
	}

	sendAlternatives()

	if !timer.Stop() {
		<-timer.C
	}

	timer.Reset(sendAlternativesTimeout)
}

func onNodeDown(addr string) {
	removeNode(addr)
	sendAlternatives()

	if !timer.Stop() {
		<-timer.C
	}

	timer.Reset(sendAlternativesTimeout)
}

func getParentAlternatives(parentID string) (alternatives map[string]*utils.Node) {
	nodeAlternativesLock.RLock()
	defer nodeAlternativesLock.RUnlock()

	alternatives = map[string]*utils.Node{}

	for _, alternative := range nodeAlternatives[parentID] {
		alternatives[alternative.ID] = alternative
	}

	return
}

func deleteDeployment(deploymentID string) (success bool, parentStatus int) {
	log.Debugf("deleting deployment %s", deploymentID)

	success = true

	parent := hTable.getParent(deploymentID)
	if parent != nil {
		deplClientAux := deplFactory.New()
		addr := parent.Addr + ":" + strconv.Itoa(deployer.Port)

		status := deplClientAux.ChildDeletedDeployment(addr, deploymentID, myself.ID)
		if status != http.StatusOK {
			log.Errorf("got status %d from child deleted deployment", status)

			success = false
			parentStatus = status

			return
		}

		pTable.decreaseParentCount(parent.ID)
		log.Debugf("decreased parent count for parent %s", parent.ID)
	}

	hTable.removeDeployment(deploymentID)
	log.Debugf("removed deployment %s from hierarchy table", deploymentID)

	log.Debugf("deleted deployment %s", deploymentID)

	return
}
