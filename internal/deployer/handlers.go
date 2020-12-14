package deployer

import (
	"encoding/json"
	"net/http"
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
	retryTimeout            = 10 * time.Second
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

	nodeLocCache *nodeLocationCache

	timer *time.Timer

	autoFactory  autonomic.ClientFactory
	archFactory  archimedes.ClientFactory
	deplFactory  deployer.ClientFactory
	schedFactory scheduler.ClientFactory
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

	autonomicClient = autoFactory.New(servers.AutonomicLocalHostPort)
	archimedesClient = archFactory.New(servers.ArchimedesLocalHostPort)
	schedulerClient = schedFactory.New(servers.SchedulerLocalHostPort)

	myAlternatives = sync.Map{}
	nodeAlternatives = map[string][]*utils.Node{}
	nodeAlternativesLock = sync.RWMutex{}
	hTable = newHierarchyTable()
	pTable = newParentsTable()

	suspectedChild = sync.Map{}
	suspectedDeployments = sync.Map{}
	children = sync.Map{}

	nodeLocCache = &nodeLocationCache{}

	timer = time.NewTimer(sendAlternativesTimeout)

	fallback = loadFallbackHostname(fallbackFilename)
	log.Debugf("loaded fallback %+v", fallback)

	simulateAlternatives()

	var (
		locationID s2.CellID
		status     int
	)

	for status != http.StatusOK {
		locationID, status = autonomicClient.GetLocation()

		time.Sleep(retryTimeout)
	}

	location = s2.CellFromCellID(locationID)

	log.Debugf("got location %s", location.ID().ToToken())

	go sendHeartbeatsPeriodically()
	go sendAlternativesPeriodically()
	go checkParentHeartbeatsPeriodically()
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
		archimedesClient.AddDeploymentNode(deploymentID, reqBody.Child, reqBody.Location, false)
	case api.Remove:
		archimedesClient.DeleteDeploymentNode(deploymentID, reqBody.Child.ID)
	}

	parent := hTable.getParent(deploymentID)
	if reqBody.TTL+1 >= maxHopslocationHorizon || parent == nil {
		return
	}

	deplClient := deplFactory.New(parent.Addr + ":" + strconv.Itoa(deployer.Port))
	log.Debugf("propagating %s location for deployments %+v to %s", reqBody.Child.ID, deploymentID, parent.ID)
	deplClient.PropagateLocationToHorizon(deploymentID, reqBody.Child, reqBody.Location, reqBody.TTL+1,
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

	autonomicClient.BlacklistNodes(deploymentID, myself.ID, childID)
}

func getDeploymentsHandler(w http.ResponseWriter, _ *http.Request) {
	deployments := hTable.getDeployments()
	internalUtils.SendJSONReplyOK(w, deployments)
}

func registerDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling register deployment request")

	var registerBody api.RegisterDeploymentRequestBody

	err := json.NewDecoder(r.Body).Decode(&registerBody)
	if err != nil {
		panic(err)
	}

	deploymentDTO := registerBody.DeploymentConfig
	deploymentID := deploymentDTO.DeploymentID

	var deploymentYAML api.DeploymentYAML

	err = yaml.Unmarshal(deploymentDTO.DeploymentYAMLBytes, &deploymentYAML)
	if err != nil {
		panic(err)
	}

	parent := hTable.getParent(deploymentID)

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

	if hTable.hasDeployment(deploymentID) {
		parentsMatch := parent != nil && deploymentDTO.Parent != nil && parentID == deploymentParentID
		parentDead := parent != nil && !pTable.hasParent(parentID)

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

	parent = hTable.getParent(deploymentID)

	parentID = nilString
	if parent != nil {
		parentID = parent.ID
	}

	log.Debugf("my parent is %s and the presented parent is %s", parentID,
		deploymentParentID)

	alreadyHadDeployment := false
	if hTable.hasDeployment(deploymentDTO.DeploymentID) {
		alreadyHadDeployment = true
		parentsMatch := parent != nil && deploymentDTO.Parent != nil && parentID == deploymentParentID
		parentDead := parent != nil && !pTable.hasParent(parentID)

		if !parentsMatch && !parentDead {
			shouldTakeChildren := len(deploymentDTO.Children) > 0 && checkChildren(parent, deploymentDTO.Children...)

			if !shouldTakeChildren {
				// after locking to add guarantee that in the meanwhile it wasn't added
				w.WriteHeader(http.StatusConflict)
				hTable.Unlock()

				return
			}

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
		d := deploymentYAMLToDeployment(&deploymentYAML, deploymentDTO.Static)
		go addDeploymentAsync(d, deploymentDTO.DeploymentID, deploymentYAML.InstanceNameFmt)
	}

	takeChildren(deploymentID, parent, deploymentDTO.Children...)
}

func takeChildren(deploymentID string, parent *utils.Node, children ...*utils.Node) {
	deplClient := deplFactory.New("")

	for _, child := range children {
		deplClient.SetHostPort(child.Addr + ":" + strconv.Itoa(deployer.Port))

		status := deplClient.WarnThatIAmParent(deploymentID, myself, parent)
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

	success, status := deleteDeployment(deploymentID)
	if !success {
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

// Function simulating lower API.
func getNodeCloserTo(locations []s2.CellID, _ int,
	excludeNodes map[string]interface{}) (closest *utils.Node, found bool) {
	closest = autonomicClient.GetClosestNode(locations, excludeNodes)
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

	status := archimedesClient.RegisterDeployment(deploymentID, deployment.Ports, myself)
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

		status = schedulerClient.StartInstance(deploymentID, deployment.Image, instanceName, deployment.Ports, i,
			deployment.Static, deployment.EnvVars, deployment.Command)
		if status != http.StatusOK {
			log.Errorf("got status code %d from scheduler", status)

			status = archimedesClient.DeleteDeployment(deploymentID)
			if status != http.StatusOK {
				log.Error("error deleting deployment that failed initializing")
			}

			hTable.removeDeployment(deploymentID)

			return
		}
	}
}

func deleteDeploymentAsync(deploymentID string) {
	autonomicClient.DeleteDeployment(deploymentID)
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

func addNode(nodeDeployerID, addr string) {
	if nodeDeployerID == "" {
		panic("error while adding node up")
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

	neighbor := utils.NewNode(nodeDeployerID, addr)

	myAlternatives.Store(nodeDeployerID, neighbor)
}

// Function simulation lower API
// Node up is only triggered for nodes that appeared one hop away.
func onNodeUp(id, addr string) {
	addNode(id, addr)
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
		client := deplFactory.New(parent.Addr + ":" + strconv.Itoa(deployer.Port))

		status := client.ChildDeletedDeployment(deploymentID, myself.ID)
		if status != http.StatusOK {
			log.Errorf("got status %d from child deleted deployment", status)

			success = false
			parentStatus = status

			return
		}

		pTable.decreaseParentCount(parent.ID)
	}

	hTable.removeDeployment(deploymentID)

	go deleteDeploymentAsync(deploymentID)

	return
}
