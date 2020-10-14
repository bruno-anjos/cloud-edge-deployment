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

	archimedesApi "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	publicUtils "github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/scheduler"
	"github.com/docker/go-connections/nat"
	log "github.com/sirupsen/logrus"

	"gopkg.in/yaml.v3"
)

type (
	typeMyAlternativesMapValue = *utils.Node
	typeChildrenMapValue       = *utils.Node

	typeSuspectedChildMapKey = string

	typeServicesLocationsValue = terminalServiceLocations

	terminalServiceLocations = *sync.Map

	typeTerminalLocationKey   = string
	typeTerminalLocationValue = *publicUtils.Location
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
	alternativesDir  = "/alternatives/"
	fallbackFilename = "fallback.txt"
	maxHopsToLookFor = 5
)

var (
	archimedesClient = archimedes.NewArchimedesClient(archimedes.DefaultHostPort)
	schedulerClient  = scheduler.NewSchedulerClient(scheduler.DefaultHostPort)
)

var (
	hostname string
	location *publicUtils.Location
	fallback string
	myself   *utils.Node

	myAlternatives       sync.Map
	nodeAlternatives     map[string][]*utils.Node
	nodeAlternativesLock sync.RWMutex

	hTable *hierarchyTable
	pTable *parentsTable

	serviceLocations       sync.Map
	addServiceLocationLock sync.Mutex

	suspectedChild       sync.Map
	suspectedDeployments sync.Map
	children             sync.Map
	childrenClient       = deployer.NewDeployerClient("")

	exploring sync.Map

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

	serviceLocations = sync.Map{}
	addServiceLocationLock = sync.Mutex{}

	exploring = sync.Map{}

	timer = time.NewTimer(sendAlternativesTimeout * time.Second)

	// TODO change this for location from lower API
	fallback = loadFallbackHostname(fallbackFilename)
	log.Debugf("loaded fallback %s", fallback)

	simulateAlternatives()

	var status int
	for status != http.StatusOK {
		location, status = hTable.autonomicClient.GetLocation()
	}

	log.Debugf("got location %f", location)

	go sendHeartbeatsPeriodically()
	go sendAlternativesPeriodically()
	go checkParentHeartbeatsPeriodically()
}

func migrateDeploymentHandler(_ http.ResponseWriter, r *http.Request) {
	log.Debugf("handling migrate request")

	serviceId := utils.ExtractPathVar(r, deploymentIdPathVar)

	var migrateDTO api.MigrateDTO
	err := json.NewDecoder(r.Body).Decode(&migrateDTO)
	if err != nil {
		panic(err)
		return
	}

	if !hTable.hasDeployment(serviceId) {
		log.Debugf("deployment %s does not exist, ignoring migration request", serviceId)
		return
	}

	deploymentChildren := hTable.getChildren(serviceId)
	origin, ok := deploymentChildren[migrateDTO.Origin]
	if !ok {
		log.Debugf("origin %s does not exist for service %s", migrateDTO.Origin, serviceId)
		return
	}

	target, ok := deploymentChildren[migrateDTO.Target]
	if !ok {
		log.Debugf("target %s does not exist for service %s", migrateDTO.Target, serviceId)
		return
	}

	if origin.Id == target.Id {
		log.Debugf("origin is the same as target (%s)", origin.Id)
		return
	}

	client := deployer.NewDeployerClient(origin.Addr + ":" + strconv.Itoa(deployer.Port))
	client.DeleteService(serviceId)

	client.SetHostPort(target.Addr)
	config := hTable.getDeploymentConfig(serviceId)
	isStatic := hTable.isStatic(serviceId)
	parent := hTable.getParent(serviceId)
	client.RegisterService(serviceId, isStatic, config, myself, parent)
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

	go attemptToExtend(deploymentId, targetAddr, nil, nil, 0, nil)
}

func shortenDeploymentFromHandler(w http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)
	targetId := utils.ExtractPathVar(r, nodeIdPathVar)

	log.Debugf("handling shorten deployment %s from %s", deploymentId, targetId)

	if !hTable.hasDeployment(deploymentId) {
		log.Debugf("deployment %s does not exist, ignoring shortening request", deploymentId)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	deploymentChildren := hTable.getChildren(deploymentId)
	_, ok := deploymentChildren[targetId]
	if !ok {
		log.Debugf("deployment %s does not have %s as its child, ignoring shortening request", deploymentId,
			targetId)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	client := deployer.NewDeployerClient(targetId)
	client.DeleteService(deploymentId)
}

func childDeletedDeploymentHandler(_ http.ResponseWriter, r *http.Request) {
	log.Debugf("handling child deleted request")
	serviceId := utils.ExtractPathVar(r, deploymentIdPathVar)
	childId := utils.ExtractPathVar(r, nodeIdPathVar)

	hTable.removeChild(serviceId, childId)
}

func getDeploymentsHandler(w http.ResponseWriter, _ *http.Request) {
	deployments := hTable.getDeployments()
	utils.SendJSONReplyOK(w, deployments)
}

func registerDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling register deployment request")

	var deploymentDTO api.DeploymentDTO
	err := json.NewDecoder(r.Body).Decode(&deploymentDTO)
	if err != nil {
		panic(err)
	}

	var deploymentYAML api.DeploymentYAML
	err = yaml.Unmarshal(deploymentDTO.DeploymentYAMLBytes, &deploymentYAML)
	if err != nil {
		panic(err)
	}

	if hTable.hasDeployment(deploymentDTO.DeploymentId) {
		w.WriteHeader(http.StatusConflict)
		return
	}

	hTable.addDeployment(&deploymentDTO)
	if deploymentDTO.Parent != nil {
		if !pTable.hasParent(deploymentDTO.Parent.Id) {
			pTable.addParent(deploymentDTO.Parent)
		}
	}

	deployment := deploymentYAMLToDeployment(&deploymentYAML, deploymentDTO.Static)

	go addDeploymentAsync(deployment, deploymentDTO.DeploymentId)
}

func deleteDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)

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

func startResolveUpTheTreeHandler(w http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)

	var reqBody api.StartResolveUpTheTreeRequestBody
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		panic(err)
	}

	parent := hTable.getParent(deploymentId)
	if parent == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	log.Debugf("starting resolution (%s) %s", deploymentId, reqBody.Host, parent.Id)

	go resolveUp(parent.Id, deploymentId, hostname, &reqBody)
}

func resolveUp(parentId, deploymentId, origin string, toResolve *archimedesApi.ToResolveDTO) {
	log.Debugf("resolving (%s) %s through %s", deploymentId, toResolve.Host, parentId)

	deplClient := deployer.NewDeployerClient(parentId + ":" + strconv.Itoa(deployer.Port))
	status := deplClient.ResolveUpTheTree(deploymentId, origin, toResolve)
	if status != http.StatusOK {
		log.Debugf("got %d while attempting to resolve up the tree", status)
	}
}

func resolveUpTheTreeHandler(w http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)

	var reqBody api.ResolveUpTheTreeRequestBody
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		panic(err)
	}

	log.Debugf("resolving (%s) %s for %s", deploymentId, reqBody.ToResolve.Host, reqBody.Origin)

	archClient := archimedes.NewArchimedesClient(publicUtils.ArchimedesServiceName + ":" + strconv.Itoa(archimedes.Port))
	rHost, rPort, status := archClient.ResolveLocally(reqBody.ToResolve.Host, reqBody.ToResolve.Port)

	archClient.SetHostPort(reqBody.Origin + ":" + strconv.Itoa(archimedes.Port))
	id := reqBody.ToResolve.Host + ":" + reqBody.ToResolve.Port.Port()

	switch status {
	case http.StatusOK:
		resolved := &archimedesApi.ResolvedDTO{
			Host: rHost,
			Port: rPort,
		}
		archClient.SetResolvingAnswer(id, resolved)
		log.Debugf("resolved (%s) %s locally to %s", deploymentId, reqBody.ToResolve.Host, resolved)
	case http.StatusNotFound:
		parent := hTable.getParent(deploymentId)
		if parent == nil {
			archClient.SetResolvingAnswer(id, nil)
		} else {
			go resolveUp(parent.Id, deploymentId, reqBody.Origin, reqBody.ToResolve)
		}
	default:
		log.Debugf("got status %d while trying to resolve locally in archimedes", status)
		w.WriteHeader(http.StatusInternalServerError)
		archClient.SetResolvingAnswer(id, nil)
	}
}

func redirectClientDownTheTreeHandler(w http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)

	var reqBody api.RedirectClientDownTheTreeRequestBody
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		panic(err)
	}

	clientLocation := reqBody

	auxChildren := hTable.getChildren(deploymentId)
	if len(auxChildren) == 0 {
		log.Debugf("no children to redirect client to")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	myLocation, status := hTable.autonomicClient.GetLocation()
	if status != http.StatusOK {
		log.Error("could not get my location")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var (
		bestDiff    = myLocation.CalcDist(clientLocation)
		bestNode    = myself.Id
		auxLocation *publicUtils.Location
	)

	autoClient := autonomic.NewAutonomicClient("")
	for id := range auxChildren {
		autoClient.SetHostPort(id + ":" + strconv.Itoa(autonomic.Port))
		auxLocation, status = autoClient.GetLocation()
		if status != http.StatusOK {
			log.Errorf("got %d while trying to get %s location", status, id)
			continue
		}

		currDiff := auxLocation.CalcDist(clientLocation)
		if currDiff < bestDiff {
			bestDiff = currDiff
			bestNode = id
		}
	}

	log.Debugf("best node in vicinity to redirect client from %+v to is %s", clientLocation, bestNode)

	value, ok := serviceLocations.Load(deploymentId)
	if ok {
		terminalLocations := value.(terminalServiceLocations)
		terminalLocations.Range(func(key, value interface{}) bool {
			nodeId := key.(typeTerminalLocationKey)
			nodeLoc := value.(typeTerminalLocationValue)

			diff := nodeLoc.CalcDist(clientLocation)
			if diff < bestDiff {
				bestNode = nodeId
				bestDiff = diff
			}

			return true
		})
	}

	if bestNode != myself.Id {
		log.Debugf("will redirect client at %f to %s", clientLocation, bestNode)

		id := deploymentId + "_" + bestNode
		_, ok = exploring.Load(id)
		if ok {
			childAutoClient := autonomic.NewAutonomicClient(bestNode + ":" + strconv.Itoa(autonomic.Port))
			status = childAutoClient.SetExploredSuccessfully(deploymentId, bestNode)
			if status != http.StatusOK {
				log.Errorf("got status %d when setting %s exploration as success", status, bestNode)
			}
			exploring.Delete(id)
		}

		var respBody api.RedirectClientDownTheTreeResponseBody
		respBody = bestNode
		utils.SendJSONReplyOK(w, respBody)
	} else {
		log.Debugf("client is already connect to closest node")
		w.WriteHeader(http.StatusNoContent)
	}
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

func terminalLocationHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)

	var reqBody api.TerminalLocationRequestBody
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		panic(err)
	}

	var terminalLocations terminalServiceLocations
	value, ok := serviceLocations.Load(deploymentId)
	if !ok {
		addServiceLocationLock.Lock()
		_, ok = serviceLocations.Load(deploymentId)
		if !ok {
			terminalLocations = &sync.Map{}
			serviceLocations.Store(deploymentId, terminalLocations)
		} else {
			terminalLocations = value.(typeServicesLocationsValue)
		}
		addServiceLocationLock.Unlock()
	} else {
		terminalLocations = value.(typeServicesLocationsValue)
	}

	log.Debugf("got terminal location %+v for deployment %s from %s", reqBody.Location, deploymentId, reqBody.Child)

	terminalLocations.Store(reqBody.Child, reqBody.Location)

	parent := hTable.getParent(deploymentId)
	if parent != nil {
		deplClient := deployer.NewDeployerClient(parent.Id + ":" + strconv.Itoa(deployer.Port))
		deplClient.SetTerminalLocation(deploymentId, myself.Id, reqBody.Location)
	}
}

func setExploringHandler(w http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)
	childId := utils.ExtractPathVar(r, nodeIdPathVar)

	ok := hTable.hasDeployment(deploymentId)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	id := deploymentId + "_" + childId
	exploring.Store(id, nil)
}

// TODO function simulating lower API
func getNodeCloserTo(location *publicUtils.Location, maxHopsToLookFor int,
	excludeNodes map[string]struct{}) (closest string, found bool) {
	closest = hTable.autonomicClient.GetClosestNode(location, excludeNodes)
	found = closest != ""
	return
}

func addDeploymentAsync(deployment *Deployment, deploymentId string) {
	log.Debugf("adding deployment %s", deploymentId)

	status := archimedesClient.RegisterService(deploymentId, deployment.Ports)
	if status != http.StatusOK {
		log.Errorf("got status code %d from archimedes", status)
		return
	}

	for i := 0; i < deployment.NumberOfInstances; i++ {
		status = schedulerClient.StartInstance(deploymentId, deployment.Image, deployment.Ports, deployment.Static,
			deployment.EnvVars)
		if status != http.StatusOK {
			log.Errorf("got status code %d from scheduler", status)

			status = archimedesClient.DeleteService(deploymentId)
			if status != http.StatusOK {
				log.Error("error deleting service that failed initializing")
			}

			hTable.removeDeployment(deploymentId)
			return
		}
	}

	hTable.setLinkOnly(deploymentId, false)
}

func deleteDeploymentAsync(deploymentId string) {
	instances, status := archimedesClient.GetService(deploymentId)
	if status != http.StatusOK {
		log.Errorf("got status %d while requesting service %s instances", status, deploymentId)
		return
	}

	status = archimedesClient.DeleteService(deploymentId)
	if status != http.StatusOK {
		log.Warnf("got status code %d from archimedes", status)
		return
	}

	for instanceId := range instances {
		status = schedulerClient.StopInstance(instanceId)
		if status != http.StatusOK {
			log.Warnf("got status code %d from scheduler", status)
			return
		}
	}
}

func deploymentYAMLToDeployment(deploymentYAML *api.DeploymentYAML, static bool) *Deployment {
	log.Debugf("%+v", deploymentYAML)

	numContainers := len(deploymentYAML.Spec.Template.Spec.Containers)
	if numContainers > 1 {
		panic("more than one container per service is not supported")
	} else if numContainers == 0 {
		panic("no container provided")
	}

	containerSpec := deploymentYAML.Spec.Template.Spec.Containers[0]

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
		DeploymentId:      deploymentYAML.Spec.ServiceName,
		NumberOfInstances: deploymentYAML.Spec.Replicas,
		Image:             containerSpec.Image,
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

// TODO function simulation lower API
// Node down is only triggered for nodes that were one hop away
func onNodeDown(id string) {
	myAlternatives.Delete(id)
	sendAlternatives()
	if !timer.Stop() {
		<-timer.C
	}
	timer.Reset(sendAlternativesTimeout * time.Second)
}

func addPortToAddr(addr string) string {
	if !strings.Contains(addr, ":") {
		return addr + ":" + strconv.Itoa(deployer.Port)
	}
	return addr
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
