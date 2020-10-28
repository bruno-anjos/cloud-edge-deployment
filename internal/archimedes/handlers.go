package archimedes

import (
	"encoding/json"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/archimedes/clients"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	publicUtils "github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type (
	redirectedConfig struct {
		Target  string
		Goal    int32
		Current int32
		Done    bool
	}

	redirectionsMapValue = *redirectedConfig
)

const (
	maxHops = 2
)

var (
	messagesReceived sync.Map
	sTable           *servicesTable
	redirectionsMap  sync.Map
	archimedesId     string
	hostname         string
	clientsManager   *clients.Manager
)

func init() {
	messagesReceived = sync.Map{}

	sTable = newServicesTable()
	redirectionsMap = sync.Map{}
	clientsManager = clients.NewManager()

	var err error
	hostname, err = os.Hostname()
	if err != nil {
		panic(err)
	}

	archimedesId = uuid.New().String()

	log.Infof("ARCHIMEDES ID: %s", archimedesId)
}

func registerServiceHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling request in registerService handler")

	serviceId := utils.ExtractPathVar(r, serviceIdPathVar)

	req := api.RegisterServiceRequestBody{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	serviceDTO := req

	service := &api.Service{
		Id:    serviceId,
		Ports: serviceDTO.Ports,
	}

	_, ok := sTable.getService(serviceId)
	if ok {
		w.WriteHeader(http.StatusConflict)
		return
	}

	newTableEntry := &api.ServicesTableEntryDTO{
		Host:         archimedesId,
		HostAddr:     archimedes.DefaultHostPort,
		Service:      service,
		Instances:    map[string]*api.Instance{},
		NumberOfHops: 0,
		MaxHops:      0,
		Version:      0,
	}

	sTable.addService(serviceId, newTableEntry)
	sendServicesTable()

	log.Debugf("added service %s", serviceId)
}

func deleteServiceHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling request in deleteService handler")

	serviceId := utils.ExtractPathVar(r, serviceIdPathVar)

	_, ok := sTable.getService(serviceId)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	sTable.deleteService(serviceId)
	redirectionsMap.Delete(serviceId)

	log.Debugf("deleted service %s", serviceId)
}

func registerServiceInstanceHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling request in registerServiceInstance handler")

	serviceId := utils.ExtractPathVar(r, serviceIdPathVar)

	_, ok := sTable.getService(serviceId)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	instanceId := utils.ExtractPathVar(r, instanceIdPathVar)

	req := api.RegisterServiceInstanceRequestBody{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	instanceDTO := req

	ok = sTable.serviceHasInstance(serviceId, instanceId)
	if ok {
		w.WriteHeader(http.StatusConflict)
		return
	}

	var host string
	if instanceDTO.Local {
		host = instanceId
	} else {
		host, _, err = net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			panic(err)
		}
	}

	instance := &api.Instance{
		Id:              instanceId,
		Ip:              host,
		ServiceId:       serviceId,
		PortTranslation: instanceDTO.PortTranslation,
		Initialized:     instanceDTO.Static,
		Static:          instanceDTO.Static,
		Local:           instanceDTO.Local,
	}

	sTable.addInstance(serviceId, instanceId, instance)
	sendServicesTable()
	log.Debugf("added instance %s to service %s", instanceId, serviceId)
}

func deleteServiceInstanceHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling request in deleteServiceInstance handler")

	serviceId := utils.ExtractPathVar(r, serviceIdPathVar)
	_, ok := sTable.getService(serviceId)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	instanceId := utils.ExtractPathVar(r, instanceIdPathVar)
	instance, ok := sTable.getServiceInstance(serviceId, instanceId)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	sTable.deleteInstance(instance.ServiceId, instanceId)

	log.Debugf("deleted instance %s from service %s", instanceId, serviceId)
}

func getAllServicesHandler(w http.ResponseWriter, _ *http.Request) {
	log.Debug("handling request in getAllServices handler")

	var resp api.GetAllServicesResponseBody
	resp = sTable.getAllServices()
	utils.SendJSONReplyOK(w, resp)
}

func getAllServiceInstancesHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling request in getAllServiceInstances handler")

	serviceId := utils.ExtractPathVar(r, serviceIdPathVar)

	_, ok := sTable.getService(serviceId)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var resp api.GetServiceResponseBody
	resp = sTable.getAllServiceInstances(serviceId)
	utils.SendJSONReplyOK(w, resp)
}

func getServiceInstanceHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling request in getServiceInstance handler")

	serviceId := utils.ExtractPathVar(r, serviceIdPathVar)
	instanceId := utils.ExtractPathVar(r, instanceIdPathVar)

	instance, ok := sTable.getServiceInstance(serviceId, instanceId)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var resp api.GetServiceInstanceResponseBody
	resp = *instance

	utils.SendJSONReplyOK(w, resp)
}

func getInstanceHandler(w http.ResponseWriter, r *http.Request) {
	instanceId := utils.ExtractPathVar(r, instanceIdPathVar)

	instance, ok := sTable.getInstance(instanceId)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var resp api.GetInstanceResponseBody
	resp = *instance
	utils.SendJSONReplyOK(w, resp)
}

func whoAreYouHandler(w http.ResponseWriter, _ *http.Request) {
	log.Debug("handling whoAreYou request")
	var resp api.WhoAreYouResponseBody
	resp = archimedesId
	utils.SendJSONReplyOK(w, resp)
}

func getServicesTableHandler(w http.ResponseWriter, _ *http.Request) {
	var resp api.GetServicesTableResponseBody
	discoverMsg := sTable.toDiscoverMsg()
	if discoverMsg != nil {
		resp = *discoverMsg
	}

	utils.SendJSONReplyOK(w, resp)
}

func resolveHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var reqBody api.ResolveRequestBody
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		log.Errorf("bad request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	defer func() {
		log.Debugf("took %f to answer request %s", time.Since(start).Seconds(), reqBody.Id)
	}()

	redirect, targetUrl := checkForRedirections(reqBody.ToResolve.Host)
	if redirect {
		log.Debugf("redirecting %s to %s to achieve load balancing", reqBody.ToResolve.Host, targetUrl.Host)
		clientsManager.RemoveFromExploring(reqBody.DeploymentId)
		http.Redirect(w, r, targetUrl.String(), http.StatusPermanentRedirect)
		return
	}

	deplClient := deployer.NewDeployerClient(publicUtils.DeployerServiceName + ":" + strconv.Itoa(deployer.Port))
	redirectTo, status := deplClient.RedirectDownTheTree(reqBody.DeploymentId, reqBody.Location)
	switch status {
	case http.StatusNoContent:
		break
	case http.StatusOK:
		log.Debugf("redirecting client from %+v", reqBody.Location)
		targetUrl = url.URL{
			Scheme: "http",
			Host:   redirectTo + ":" + strconv.Itoa(archimedes.Port),
			Path:   api.GetResolvePath(),
		}
		clientsManager.RemoveFromExploring(reqBody.DeploymentId)
		http.Redirect(w, r, targetUrl.String(), http.StatusPermanentRedirect)
		return
	default:
		log.Errorf("got status %d while redirecting down the tree", status)
		return
	}

	resolved, found := resolveLocally(reqBody.ToResolve)
	if !found {
		var fallback string
		fallback, status = deplClient.GetFallback()
		if status != http.StatusOK {
			log.Errorf("got status %d while asking for fallback from deployer", fallback)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		fallbackURL := url.URL{
			Scheme: "http",
			Host:   fallback + ":" + strconv.Itoa(archimedes.Port),
			Path:   api.GetResolvePath(),
		}
		clientsManager.RemoveFromExploring(reqBody.DeploymentId)
		http.Redirect(w, r, fallbackURL.String(), http.StatusPermanentRedirect)
		return
	}

	clientsManager.UpdateNumRequests(reqBody.DeploymentId, reqBody.Location)

	var resp api.ResolveResponseBody
	resp = *resolved

	clientsManager.RemoveFromExploring(reqBody.DeploymentId)
	utils.SendJSONReplyOK(w, resp)
}

func resolveLocallyHandler(w http.ResponseWriter, r *http.Request) {
	var req api.ResolveLocallyRequestBody
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		panic(err)
	}

	toResolve := &req
	resolved, found := resolveLocally(toResolve)
	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	utils.SendJSONReplyOK(w, resolved)
}

func checkForRedirections(hostToResolve string) (redirect bool, targetUrl url.URL) {
	redirect = false

	value, ok := redirectionsMap.Load(hostToResolve)
	if ok {
		redirectConfig := value.(redirectionsMapValue)
		if !redirectConfig.Done {
			reachedGoal := atomic.CompareAndSwapInt32(&redirectConfig.Current, redirectConfig.Goal-1,
				redirectConfig.Goal)
			redirect, targetUrl = true, url.URL{
				Scheme: "http",
				Host:   redirectConfig.Target + ":" + strconv.Itoa(archimedes.Port),
				Path:   api.GetResolvePath(),
			}
			if reachedGoal {
				log.Debugf("completed goal of redirecting %d clients to %s for service %s", redirectConfig.Target,
					redirectConfig.Goal, hostToResolve)
				redirectConfig.Done = true
			}
			return
		}
	}

	return
}

func resolveLocally(toResolve *api.ToResolveDTO) (resolved *api.ResolvedDTO, found bool) {
	found = false

	service, sOk := sTable.getService(toResolve.Host)
	if !sOk {
		instance, iOk := sTable.getInstance(toResolve.Host)
		if !iOk {
			return
		}

		resolved, found = resolveInstance(toResolve.Port, instance)
		return
	}

	instances := sTable.getAllServiceInstances(service.Id)

	if len(instances) == 0 {
		log.Debugf("no instances for service %s", service.Id)
		return
	}

	var randInstance *api.Instance
	randNum := rand.Intn(len(instances))
	for _, instance := range instances {
		if randNum == 0 {
			randInstance = instance
		} else {
			randNum--
		}
	}

	resolved, found = resolveInstance(toResolve.Port, randInstance)
	if found {
		log.Debugf("resolved %s:%s to %s:%s", toResolve.Host, toResolve.Port.Port(), resolved.Host, resolved.Port)
	}

	return
}

func discoverHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling request in discoverService handler")

	req := api.DiscoverRequestBody{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Error(err)
		return
	}

	discoverMsg := req

	_, ok := messagesReceived.Load(discoverMsg.MessageId)
	if ok {
		log.Debugf("repeated message %s, ignoring...", discoverMsg.MessageId)
		return
	}

	log.Debugf("got discover message %+v", discoverMsg)

	remoteAddr, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		panic(err)
	}

	preprocessMessage(remoteAddr, &discoverMsg)

	sTable.updateTableWithDiscoverMessage(discoverMsg.NeighborSent, &discoverMsg)

	messagesReceived.Store(discoverMsg.MessageId, struct{}{})

	postprocessMessage(&discoverMsg)
	broadcastMsgWithHorizon(&discoverMsg, maxHops)
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	serviceId := utils.ExtractPathVar(r, serviceIdPathVar)

	var req api.RedirectRequestBody
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Debugf("redirecting %d clients to %s", req.Amount, req.Target)

	redirectConfig := &redirectedConfig{
		Target:  req.Target,
		Goal:    req.Amount,
		Current: 0,
		Done:    false,
	}

	redirectionsMap.Store(serviceId, redirectConfig)
}

func removeRedirectionHandler(_ http.ResponseWriter, r *http.Request) {
	serviceId := utils.ExtractPathVar(r, serviceIdPathVar)
	redirectionsMap.Delete(serviceId)
}

func getRedirectedHandler(w http.ResponseWriter, r *http.Request) {
	serviceId := utils.ExtractPathVar(r, serviceIdPathVar)

	value, ok := redirectionsMap.Load(serviceId)
	if !ok {
		log.Debugf("service %s is not being redirected", serviceId)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	redirected := value.(redirectionsMapValue)

	utils.SendJSONReplyOK(w, redirected.Current)
}

func setExploringClientLocationHandler(_ http.ResponseWriter, r *http.Request) {
	serviceId := utils.ExtractPathVar(r, serviceIdPathVar)

	var reqBody api.SetExploringClientLocationRequestBody
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		panic(err)
	}

	log.Debugf("set exploring location %v for service %s", reqBody, serviceId)

	clientsManager.AddToExploring(serviceId, reqBody)
}

// TODO simulating
func getLoadHandler(w http.ResponseWriter, r *http.Request) {
	serviceId := utils.ExtractPathVar(r, serviceIdPathVar)

	load := clientsManager.GetLoad(serviceId)
	log.Debugf("got load %d for service %s", load, serviceId)

	utils.SendJSONReplyOK(w, load)
}

func getClientCentroidsHandler(w http.ResponseWriter, r *http.Request) {
	deploymentId := utils.ExtractPathVar(r, serviceIdPathVar)
	centroids, ok := clientsManager.GetDeploymentClientsCentroids(deploymentId)
	if !ok {
		w.WriteHeader(http.StatusNoContent)
	} else {
		utils.SendJSONReplyOK(w, centroids)
	}
}

func preprocessMessage(remoteAddr string, discoverMsg *api.DiscoverMsg) {
	for _, entry := range discoverMsg.Entries {
		if entry.Host == discoverMsg.NeighborSent {
			entry.HostAddr = remoteAddr
			for _, instance := range entry.Instances {
				instance.Ip = remoteAddr
			}
		}
	}
}

func postprocessMessage(discoverMsg *api.DiscoverMsg) {
	var servicesToDelete []string

	for serviceId, entry := range discoverMsg.Entries {
		if entry.NumberOfHops > maxHops {
			servicesToDelete = append(servicesToDelete, serviceId)
		}
	}

	for _, serviceToDelete := range servicesToDelete {
		delete(discoverMsg.Entries, serviceToDelete)
	}
}

func sendServicesTable() {
	discoverMsg := sTable.toChangedDiscoverMsg()
	if discoverMsg == nil {
		return
	}

	broadcastMsgWithHorizon(discoverMsg, maxHops)
}

func resolveInstance(originalPort nat.Port, instance *api.Instance) (*api.ResolvedDTO, bool) {
	portNatResolved, ok := instance.PortTranslation[originalPort]
	if !ok {
		return nil, false
	}

	return &api.ResolvedDTO{
		Host: hostname,
		Port: portNatResolved[0].HostPort,
	}, true
}

func broadcastMsgWithHorizon(discoverMsg *api.DiscoverMsg, hops int) {
	// TODO this simulates the lower level layer
	return
}
