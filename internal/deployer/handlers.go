package deployer

import (
	"encoding/json"
	"errors"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/scheduler"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
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
	extendAttemptTimeout    = 10
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
	fallback string
	location string
	myself   *utils.Node

	httpClient *http.Client

	myAlternatives       sync.Map
	nodeAlternatives     map[string][]*utils.Node
	nodeAlternativesLock sync.RWMutex

	hierarchyTable *HierarchyTable
	parentsTable   *ParentsTable

	suspectedChild sync.Map
	children       sync.Map
	childrenClient = deployer.NewDeployerClient("")

	timer *time.Timer
)

func init() {
	aux, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	hostname = aux + ":" + strconv.Itoa(deployer.Port)

	deployerId := uuid.New()
	myself = utils.NewNode(deployerId.String(), hostname)

	log.Debugf("DEPLOYER_ID: %s", deployerId)

	httpClient = &http.Client{
		Timeout: 10 * time.Second,
	}

	myAlternatives = sync.Map{}
	nodeAlternatives = map[string][]*utils.Node{}
	nodeAlternativesLock = sync.RWMutex{}
	hierarchyTable = NewHierarchyTable()
	parentsTable = NewParentsTable()

	suspectedChild = sync.Map{}
	children = sync.Map{}

	timer = time.NewTimer(sendAlternativesTimeout * time.Second)

	// TODO change this for location from lower API
	location = ""
	fallback = loadFallbackHostname(fallbackFilename)

	simulateAlternatives()

	go sendHeartbeatsPeriodically()
	go sendAlternativesPeriodically()
	go checkParentHeartbeatsPeriodically()
}

func expandTreeHandler(_ http.ResponseWriter, r *http.Request) {
	log.Debugf("handling expand tree request")

	deploymentId := utils.ExtractPathVar(r, DeploymentIdPathVar)

	log.Debugf("quality not assured for %s", deploymentId)

	reqLocation := ""
	err := json.NewDecoder(r.Body).Decode(&location)
	if err != nil {
		panic(err)
	}

	go attemptToExtend(deploymentId, nil, reqLocation, maxHopsToLookFor)
}

func migrateDeploymentHandler(_ http.ResponseWriter, r *http.Request) {
	log.Debugf("handling migrate request")

	serviceId := utils.ExtractPathVar(r, DeploymentIdPathVar)

	var migrateDTO api.MigrateDTO
	err := json.NewDecoder(r.Body).Decode(&migrateDTO)
	if err != nil {
		panic(err)
		return
	}

	if !hierarchyTable.HasDeployment(serviceId) {
		log.Debugf("deployment %s does not exist, ignoring migration request", serviceId)
		return
	}

	deploymentChildren := hierarchyTable.GetChildren(serviceId)
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

	client := deployer.NewDeployerClient(origin.Addr + ":" + strconv.Itoa(deployer.Port))
	client.DeleteService(serviceId)

	client.SetHostPort(target.Addr)
	config := hierarchyTable.GetDeploymentConfig(serviceId)
	isStatic := hierarchyTable.IsStatic(serviceId)
	client.RegisterService(serviceId, isStatic, config)
}

func extendDeploymentToHandler(w http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, DeploymentIdPathVar)
	targetId := utils.ExtractPathVar(r, DeployerIdPathVar)

	log.Debugf("handling request to extend deployment %s to %s", deploymentId, targetId)

	if !hierarchyTable.HasDeployment(deploymentId) {
		log.Debugf("deployment %s does not exist, ignoring extension request", deploymentId)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	deploymentChildren := hierarchyTable.GetChildren(deploymentId)
	_, ok := deploymentChildren[targetId]
	if !ok {
		log.Debugf("deployment %s does not have %s as its child, ignoring extension request", deploymentId,
			targetId)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	config := hierarchyTable.GetDeploymentConfig(deploymentId)

	client := deployer.NewDeployerClient(targetId)
	isStatic := hierarchyTable.IsStatic(deploymentId)
	client.RegisterService(deploymentId, isStatic, config)
}

func shortenDeploymentFromHandler(w http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, DeploymentIdPathVar)
	targetId := utils.ExtractPathVar(r, DeployerIdPathVar)

	log.Debugf("handling shorten deployment %s from %s", deploymentId, targetId)

	if !hierarchyTable.HasDeployment(deploymentId) {
		log.Debugf("deployment %s does not exist, ignoring shortening request", deploymentId)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	deploymentChildren := hierarchyTable.GetChildren(deploymentId)
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
	serviceId := utils.ExtractPathVar(r, DeploymentIdPathVar)
	childId := utils.ExtractPathVar(r, DeployerIdPathVar)

	hierarchyTable.RemoveChild(serviceId, childId)
}

func getDeploymentsHandler(w http.ResponseWriter, _ *http.Request) {
	deployments := hierarchyTable.GetDeployments()
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

	if hierarchyTable.HasDeployment(deploymentDTO.DeploymentId) {
		w.WriteHeader(http.StatusConflict)
		return
	}

	hierarchyTable.AddDeployment(&deploymentDTO)

	deployment := deploymentYAMLToDeployment(&deploymentYAML, deploymentDTO.Static)

	go addDeploymentAsync(deployment, deploymentDTO.DeploymentId)
}

func deleteDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, DeploymentIdPathVar)

	parent := hierarchyTable.GetParent(deploymentId)
	if parent != nil {
		client := deployer.NewDeployerClient(parent.Addr + ":" + strconv.Itoa(deployer.Port))
		status := client.ChildDeletedDeployment(deploymentId, myself.Id)
		if status != http.StatusOK {
			log.Errorf("got status %d from child deleted deployment", status)
			w.WriteHeader(status)
			return
		}
		parentsTable.DecreaseParentCount(parent.Id)
	}

	hierarchyTable.RemoveDeployment(deploymentId)

	go deleteDeploymentAsync(deploymentId)
}

func whoAreYouHandler(w http.ResponseWriter, _ *http.Request) {
	utils.SendJSONReplyOK(w, myself.Id)
}

func addNodeHandler(_ http.ResponseWriter, r *http.Request) {
	var nodeAddr string
	err := json.NewDecoder(r.Body).Decode(&nodeAddr)
	if err != nil {
		panic(err)
	}

	onNodeUp(nodeAddr)
}

// TODO function simulating lower API
func getNodeCloserTo(location string, maxHopsToLookFor int, excludeNodes map[string]struct{}) string {
	var (
		alternatives []*utils.Node
	)

	myAlternatives.Range(func(key, value interface{}) bool {
		node := value.(typeMyAlternativesMapValue)
		if _, ok := excludeNodes[node.Id]; ok {
			return true
		}
		alternatives = append(alternatives, node)
		return true
	})

	if len(alternatives) == 0 {
		return ""
	}

	randIdx := rand.Intn(len(alternatives))
	return alternatives[randIdx].Addr
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

			hierarchyTable.RemoveDeployment(deploymentId)
			return
		}
	}

	hierarchyTable.SetLinkOnly(deploymentId, false)
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

func getDeployerIdFromAddr(addr string) (string, error) {
	var nodeDeployerId string

	otherDeployerAddr := addPortToAddr(addr)

	req := utils.BuildRequest(http.MethodGet, otherDeployerAddr, api.GetWhoAreYouPath(), nil)
	status, _ := utils.DoRequest(httpClient, req, &nodeDeployerId)

	if status != http.StatusOK {
		log.Errorf("got status code %d from other deployer", status)
		return "", errors.New("got status code %d from other deployer")
	}

	return nodeDeployerId, nil
}

func addNode(nodeDeployerId, addr string) bool {
	if nodeDeployerId == "" {
		var err error
		nodeDeployerId, err = getDeployerIdFromAddr(addr)
		if err != nil {
			return false
		}
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
	addNode("", addr)
	sendAlternatives()
	timer.Reset(sendAlternativesTimeout * time.Second)
}

// TODO function simulation lower API
// Node down is only triggered for nodes that were one hop away
func onNodeDown(addr string) {
	id, err := getDeployerIdFromAddr(addr)
	if err != nil {
		return
	}

	myAlternatives.Delete(id)
	sendAlternatives()
	timer.Reset(sendAlternativesTimeout * time.Second)
}

func addPortToAddr(addr string) string {
	if !strings.Contains(addr, ":") {
		return addr + ":" + strconv.Itoa(deployer.Port)
	}
	return addr
}
