package archimedes

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"

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
	resolvingInTree  sync.Map
	resolvedInTree   sync.Map
	waitingChannels  sync.Map
)

func init() {
	messagesReceived = sync.Map{}

	sTable = newServicesTable()
	redirectionsMap = sync.Map{}
	resolvingInTree = sync.Map{}
	resolvedInTree = sync.Map{}
	waitingChannels = sync.Map{}

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
	log.Debugf("handling resolve request")

	var reqBody api.ResolveRequestBody
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	redirect, targetUrl := checkForRedirections(reqBody.ToResolve.Host)
	if redirect {
		http.Redirect(w, r, targetUrl.String(), http.StatusPermanentRedirect)
	}

	deplClient := deployer.NewDeployerClient(utils.DeployerServiceName + ":" + strconv.Itoa(deployer.Port))
	redirectTo, status := deplClient.RedirectDownTheTree(reqBody.DeploymentId, reqBody.Location)
	switch status {
	case http.StatusNoContent:
		break
	case http.StatusOK:
		targetUrl = url.URL{
			Scheme: "http",
			Host:   redirectTo + ":" + strconv.Itoa(archimedes.Port),
			Path:   api.GetResolvePath(),
		}
		http.Redirect(w, r, targetUrl.String(), http.StatusPermanentRedirect)
		return
	default:
		log.Errorf("got status %d while redirecting down the tree", status)
		return
	}

	resolved, found := resolveLocally(reqBody.ToResolve)
	if !found {
		// TODO Redirect to fallback
	}

	var resp api.ResolveResponseBody
	resp = *resolved

	utils.SendJSONReplyOK(w, resp)
}

func resolveInTree(id, deploymentId string, toResolve *api.ToResolveDTO) (resolved *api.ResolvedDTO) {
	log.Debugf("resolving (%s) %s up the tree", deploymentId, toResolve.Host)

	_, ok := resolvingInTree.Load(id)
	if ok {
		log.Debugf("already resolving (%s) %s", deploymentId, toResolve.Host)
		// If there is already a resolution for this given id ocurring
		value, cOk := waitingChannels.Load(id)
		if !cOk {
			panic(fmt.Sprintf("there should be a waiting channel for %s", id))
		}

		// Get the channel and wait for it to be complete
		waitChan := value.(chan struct{})
		<-waitChan

		value, ok = resolvedInTree.Load(id)
		if !ok {
			panic(fmt.Sprintf("value for %s was supposed to be in map", id))
		}

		if value == nil {
			resolved = nil
			log.Debugf("resolved (%s) %s to nil", deploymentId, toResolve.Host)
		} else {
			resolved = value.(*api.ResolvedDTO)
			log.Debugf("resolved (%s) %s to %s", deploymentId, toResolve.Host, resolved.Host)
		}

		return
	}

	value, ok := resolvedInTree.Load(id)
	if ok {
		// there was no resolution occurring and the value had been resolved before
		resolved = value.(*api.ResolvedDTO)
		return
	}

	// there was no resolution occurring and it had'nt been resolved before
	waitChan := make(chan struct{})
	waitingChannels.Store(id, waitChan)
	resolvingInTree.Store(id, nil)
	deplClient := deployer.NewDeployerClient(utils.DeployerServiceName + ":" + strconv.Itoa(deployer.Port))
	status := deplClient.StartResolveUpTheTree(deploymentId, toResolve)
	if status != http.StatusOK {
		log.Debugf("got %d while attempting to start resolving (%s) %s", status, deploymentId, toResolve.Host)
		return nil
	}
	<-waitChan

	value, ok = resolvingInTree.Load(id)
	if !ok {
		panic(fmt.Sprintf("value for %s was supposed to be in map", id))
	}

	if value == nil {
		resolved = nil
		log.Debugf("resolved (%s) %s to nil", deploymentId, toResolve.Host)
	} else {
		resolved = value.(*api.ResolvedDTO)
		log.Debugf("resolved (%s) %s to %s", deploymentId, toResolve.Host, resolved.Host)
	}

	return
}

func setResolutionAnswerHandler(w http.ResponseWriter, r *http.Request) {
	var reqBody api.SetResolutionAnswerRequestBody
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Debugf("got answer %s for %s", reqBody.Resolved.Host, reqBody.Id)

	value, ok := waitingChannels.Load(reqBody.Id)
	if !ok {
		log.Debugf("got answer for resolution of %s, but i wasnt waiting", reqBody.Id)
		return
	}

	waitChan := value.(chan struct{})
	resolvedInTree.Store(reqBody.Id, &reqBody.Resolved)
	resolvingInTree.Delete(reqBody.Id)
	waitingChannels.Delete(reqBody.Id)
	close(waitChan)
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
	if instance.Local {
		return &api.ResolvedDTO{
			Host: instance.Id,
			Port: originalPort.Port(),
		}, true
	} else {
		portNatResolved, ok := instance.PortTranslation[originalPort]
		if !ok {
			return nil, false
		}

		return &api.ResolvedDTO{
			Host: instance.Ip,
			Port: portNatResolved[0].HostPort,
		}, true
	}
}

func broadcastMsgWithHorizon(discoverMsg *api.DiscoverMsg, hops int) {
	// TODO this simulates the lower level layer
	return
}
