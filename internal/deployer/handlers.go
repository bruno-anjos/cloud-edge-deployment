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
	api "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/scheduler"
	publicUtils "github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/docker/go-connections/nat"
	"github.com/golang/geo/s2"
	log "github.com/sirupsen/logrus"

	"gopkg.in/yaml.v3"
)

type (
	typeMyAlternativesMapValue = *utils.Node
	typeChildrenMapValue       = *utils.Node

	typeSuspectedChildMapKey = string

	typeDeploymentsLocationsValue = terminalDeploymentLocations

	terminalDeploymentLocations = *sync.Map

	typeTerminalLocationKey   = string
	typeTerminalLocationValue = s2.CellID

	typeExploringValue = *sync.Once

	typeNodeLocationCache = s2.CellID
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

	maxHopslocationHorizon = 3
)

var (
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

	deploymentLocations       sync.Map
	addDeploymentLocationLock sync.Mutex

	suspectedChild       sync.Map
	suspectedDeployments sync.Map
	children             sync.Map
	childrenClient       = deployer.NewDeployerClient("")

	nodeLocationCache sync.Map

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

	deploymentLocations = sync.Map{}
	addDeploymentLocationLock = sync.Mutex{}

	nodeLocationCache = sync.Map{}

	exploring = sync.Map{}

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
		locationId, status = hTable.autonomicClient.GetLocation()
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

	log.Debugf("got location from %s for deployment %s", reqBody.ChildId, deploymentId)

	var nodeLocations typeDeploymentsLocationsValue
	value, ok := deploymentLocations.Load(deploymentId)
	if !ok {
		addDeploymentLocationLock.Lock()
		value, ok = deploymentLocations.Load(deploymentId)
		if !ok {
			nodeLocations = &sync.Map{}
			deploymentLocations.Store(deploymentId, nodeLocations)
		} else {
			nodeLocations = value.(typeDeploymentsLocationsValue)
		}
		addDeploymentLocationLock.Unlock()
	} else {
		nodeLocations = value.(typeDeploymentsLocationsValue)
	}
	nodeLocations.Store(reqBody.ChildId, reqBody.Location)

	if reqBody.TTL+1 > maxHopslocationHorizon {
		return
	}

	parent := hTable.getParent(deploymentId)
	deplClient := deployer.NewDeployerClient(parent.Id + ":" + strconv.Itoa(deployer.Port))
	log.Debugf("propagating %s location for deployments %+v to %s", reqBody.ChildId, deploymentId, parent.Id)
	deplClient.PropagateLocationToHorizon(deploymentId, reqBody.ChildId, reqBody.Location, reqBody.TTL+1)
}

func migrateDeploymentHandler(_ http.ResponseWriter, r *http.Request) {
	log.Debugf("handling migrate request")

	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)

	var migrateDTO api.MigrateDTO
	err := json.NewDecoder(r.Body).Decode(&migrateDTO)
	if err != nil {
		panic(err)
		return
	}

	if !hTable.hasDeployment(deploymentId) {
		log.Debugf("deployment %s does not exist, ignoring migration request", deploymentId)
		return
	}

	deploymentChildren := hTable.getChildren(deploymentId)
	origin, ok := deploymentChildren[migrateDTO.Origin]
	if !ok {
		log.Debugf("origin %s does not exist for deployment %s", migrateDTO.Origin, deploymentId)
		return
	}

	target, ok := deploymentChildren[migrateDTO.Target]
	if !ok {
		log.Debugf("target %s does not exist for deployment %s", migrateDTO.Target, deploymentId)
		return
	}

	if origin.Id == target.Id {
		log.Debugf("origin is the same as target (%s)", origin.Id)
		return
	}

	client := deployer.NewDeployerClient(origin.Addr + ":" + strconv.Itoa(deployer.Port))
	client.DeleteDeployment(deploymentId)

	client.SetHostPort(target.Addr)
	config := hTable.getDeploymentConfig(deploymentId)
	isStatic := hTable.isStatic(deploymentId)
	client.RegisterDeployment(deploymentId, isStatic, config, myself, nil)
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

	go attemptToExtend(deploymentId, targetAddr, reqBody.Location, reqBody.Children, reqBody.Parent, 0, nil, reqBody.Exploring)
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

	client := deployer.NewDeployerClient(targetId + ":" + strconv.Itoa(deployer.Port))
	client.DeleteDeployment(deploymentId)
}

func childDeletedDeploymentHandler(_ http.ResponseWriter, r *http.Request) {
	log.Debugf("handling child deleted request")
	deploymentId := utils.ExtractPathVar(r, deploymentIdPathVar)
	childId := utils.ExtractPathVar(r, nodeIdPathVar)

	hTable.removeChild(deploymentId, childId)
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

	var missingChildren []string
	deplChildren := hTable.getChildren(deploymentDTO.DeploymentId)
	for _, child := range deploymentDTO.Children {
		_, ok := deplChildren[child.Id]
		if !ok {
			missingChildren = append(missingChildren, child.Id)
		}
	}

	if hTable.hasDeployment(deploymentDTO.DeploymentId) && len(missingChildren) == 0 {
		w.WriteHeader(http.StatusConflict)
		return
	}

	for _, child := range missingChildren {
		if deploymentDTO.Parent != nil && child != deploymentDTO.Parent.Id {
			log.Debugf("can take child %s, my parent is %s", child, deploymentDTO.Parent.Id)
		} else {
			log.Debugf("rejecting child %s", child)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	if deploymentDTO.Parent != nil {
		parent := hTable.getParent(deploymentDTO.DeploymentId)
		if parent == nil || parent.Id == deploymentDTO.Parent.Id {
			log.Debugf("can take %s as parent", deploymentDTO.Parent.Id)
		} else {
			log.Debugf("rejecting parent %s", deploymentDTO.Parent.Id)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	hTable.addDeployment(&deploymentDTO)
	if deploymentDTO.Parent != nil {
		if !pTable.hasParent(deploymentDTO.Parent.Id) {
			pTable.addParent(deploymentDTO.Parent)
		}
	}

	deployment := deploymentYAMLToDeployment(&deploymentYAML, deploymentDTO.Static)

	go addDeploymentAsync(deployment, deploymentDTO.DeploymentId)

	deplClient := deployer.NewDeployerClient("")
	if deploymentDTO.Parent != nil {
		deplClient.SetHostPort(deploymentDTO.Parent.Id + ":" + strconv.Itoa(deployer.Port))
		grandparent, status := deplClient.WarnThatIAmChild(deploymentDTO.DeploymentId, myself)
		if status != http.StatusOK {
			log.Errorf("got status %d while telling %s that im his child for deployment %s", status,
				deploymentDTO.Parent.Id, deploymentDTO.DeploymentId)
			return
		}
		hTable.setDeploymentGrandparent(deploymentDTO.DeploymentId, grandparent)
	}

	for _, child := range deploymentDTO.Children {
		deplClient.SetHostPort(child.Id + ":" + strconv.Itoa(deployer.Port))
		status := deplClient.WarnThatIAmParent(deploymentDTO.DeploymentId, myself, deploymentDTO.Parent)
		if status != http.StatusOK {
			log.Errorf("got status code %d while telling %s that im his parent for %s", status, child.Id,
				deploymentDTO.DeploymentId)
		}
		hTable.addChild(deploymentDTO.DeploymentId, child)
	}

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

	var (
		bestDiff      = location.DistanceToCell(s2.CellFromCellID(clientLocation))
		bestNode      = myself.Id
		autoClient    *autonomic.Client
		auxLocationId s2.CellID
		status        int
	)

	for id := range auxChildren {
		value, ok := nodeLocationCache.Load(id)
		if !ok {
			if autoClient == nil {
				autoClient = autonomic.NewAutonomicClient(id + ":" + strconv.Itoa(autonomic.Port))
			} else {
				autoClient.SetHostPort(id + ":" + strconv.Itoa(autonomic.Port))
			}
			auxLocationId, status = autoClient.GetLocation()
			if status != http.StatusOK {
				log.Errorf("got %d while trying to get %s location", status, id)
				continue
			}
			nodeLocationCache.Store(id, auxLocationId)
		} else {
			auxLocationId = value.(typeNodeLocationCache)
		}

		auxLocation := s2.CellFromCellID(auxLocationId)
		currDiff := auxLocation.DistanceToCell(s2.CellFromCellID(clientLocation))
		if currDiff < bestDiff {
			bestDiff = currDiff
			bestNode = id
		}
	}

	log.Debugf("best node in vicinity to redirect client from %+v to is %s", clientLocation, bestNode)

	value, ok := deploymentLocations.Load(deploymentId)
	if ok {
		terminalLocations := value.(terminalDeploymentLocations)
		terminalLocations.Range(func(key, value interface{}) bool {
			nodeId := key.(typeTerminalLocationKey)
			nodeLocId := value.(typeTerminalLocationValue)
			nodeLoc := s2.CellFromCellID(nodeLocId)
			diff := nodeLoc.DistanceToCell(s2.CellFromCellID(clientLocation))
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
		value, ok = exploring.Load(id)
		if ok {
			exploring.Delete(id)
			once := value.(typeExploringValue)
			once.Do(func() {
				childAutoClient := autonomic.NewAutonomicClient(myself.Id + ":" + strconv.Itoa(autonomic.Port))
				status = childAutoClient.SetExploredSuccessfully(deploymentId, bestNode)
				if status != http.StatusOK {
					log.Errorf("got status %d when setting %s exploration as success", status, bestNode)
				}
			})
		}

		var respBody api.RedirectClientDownTheTreeResponseBody
		respBody = bestNode
		utils.SendJSONReplyOK(w, respBody)
	} else {
		log.Debugf("client is already connected to closest node")
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

// TODO function simulating lower API
func getNodeCloserTo(location s2.CellID, maxHopsToLookFor int,
	excludeNodes map[string]interface{}) (closest string, found bool) {
	closest = hTable.autonomicClient.GetClosestNode(location, excludeNodes)
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
			deployment.EnvVars)
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
	instances, status := archimedesClient.GetDeployment(deploymentId)
	if status != http.StatusOK {
		log.Errorf("got status %d while requesting deployment %s instances", status, deploymentId)
		return
	}

	status = archimedesClient.DeleteDeployment(deploymentId)
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
		panic("more than one container per deployment is not supported")
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
		DeploymentId:      deploymentYAML.Spec.DeploymentName,
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

func getParentAlternatives(parentId string) (alternatives map[string]*utils.Node) {
	nodeAlternativesLock.RLock()
	defer nodeAlternativesLock.RUnlock()

	alternatives = map[string]*utils.Node{}

	for _, alternative := range nodeAlternatives[parentId] {
		alternatives[alternative.Id] = alternative
	}

	return
}

func getDeploymentTerminalLocations(deploymentId string) []s2.CellID {
	value, ok := deploymentLocations.Load(deploymentId)
	if !ok {
		log.Debugf("no terminal locations for deployment %s", deploymentId)
		return nil
	}

	deploymentLocations := value.(typeDeploymentsLocationsValue)
	var locations []s2.CellID
	deploymentLocations.Range(func(key, value interface{}) bool {
		childLocation := value.(typeTerminalLocationValue)
		locations = append(locations, childLocation)
		return true
	})

	return locations
}

func removeTerminalLocsForChild(deploymentId, childId string) {
	value, ok := deploymentLocations.Load(deploymentId)
	if !ok {
		return
	}

	deploymentLocs := value.(typeDeploymentsLocationsValue)
	deploymentLocs.Delete(childId)
}
