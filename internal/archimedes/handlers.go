package archimedes

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/archimedes/clients"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/servers"
	internalUtils "github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"

	"github.com/golang/geo/s2"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	demmonAPI "github.com/bruno-anjos/cloud-edge-deployment/api/demmon"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	client "github.com/nm-morais/demmon-client/pkg"
	"github.com/nm-morais/demmon-common/body_types"
	log "github.com/sirupsen/logrus"
)

type (
	redirectedConfig struct {
		Target  string
		Goal    int32
		Current int32
		Done    bool
	}

	redirectingToMeMapValue = *sync.Map

	redirectionsMapValue = *redirectedConfig

	tableMessageWithHost struct {
		DiscoverMsg *api.DiscoverMsg
		Host        *utils.Node
	}
)

const (
	maxHops = 2

	daemonPort     = 8090
	requestTimeout = 5 * time.Second
	connectTimeout = 5 * time.Second

	broadcastTimeout = 10 * time.Second

	retryTimeout = 2 * time.Second
)

var (
	sTable              *deploymentsTable
	remoteTable         *remoteDeploymentsTable
	redirectServicesMap sync.Map
	redirectingToMe     sync.Map
	clientsManager      *clients.Manager

	redirectTargets *nodesPerDeployment
	exploringNodes  *explorersPerDeployment

	archimedesID string
	myself       *utils.Node
	myLocation   s2.Cell

	autoFactory autonomic.ClientFactory
	deplFactory deployer.ClientFactory

	demCli *client.DemmonClient

	fallback *utils.Node
)

func InitServer(autoFactoryAux autonomic.ClientFactory, deplFactoryAux deployer.ClientFactory) {
	sTable = newDeploymentsTable()
	remoteTable = newRemoteDeploymentsTable()

	clientsManager = clients.NewManager()

	myself = utils.NodeFromEnv()

	autoFactory = autoFactoryAux
	deplFactory = deplFactoryAux

	locationToken, ok := os.LookupEnv(utils.LocationEnvVarName)
	if !ok {
		log.Panic("location env not set")
	}

	myLocation = s2.CellFromCellID(s2.CellIDFromToken(locationToken))

	redirectTargets = &nodesPerDeployment{}
	exploringNodes = &explorersPerDeployment{}

	deplClient := deplFactory.New()

	for {
		var status int

		fallback, status = deplClient.GetFallback(servers.DeployerLocalHostPort)
		if status == http.StatusOK {
			break
		}

		time.Sleep(retryTimeout)
	}

	archimedesID = uuid.New().String()

	log.Infof("ARCHIMEDES ID: %s", archimedesID)

	demCliConf := client.DemmonClientConf{
		DemmonPort:     daemonPort,
		DemmonHostAddr: myself.Addr,
		RequestTimeout: requestTimeout,
	}

	demCli = client.New(demCliConf)
	err, errChan := demCli.ConnectTimeout(connectTimeout)
	if err != nil {
		log.Panic(err)
	}

	go internalUtils.PanicOnErrFromChan(errChan)

	msgChan, _, err := demCli.InstallBroadcastMessageHandler(demmonAPI.DiscoverMessageID)
	if err != nil {
		log.Panic(err)
	}

	go broadcastPeriodically()
	go handleBroadcastMessages(msgChan)
}

func broadcastPeriodically() {
	ticker := time.NewTicker(broadcastTimeout)

	for {
		<-ticker.C

		discMsg := sTable.toChangedDiscoverMsg()
		if discMsg == nil {
			continue
		}

		log.Debugf("broadcasting %+v", discMsg)

		err := demCli.BroadcastMessage(body_types.Message{
			ID:  demmonAPI.DiscoverMessageID,
			TTL: maxHops,
			Content: tableMessageWithHost{
				DiscoverMsg: discMsg,
				Host:        myself,
			},
		})
		if err != nil {
			log.Panic(err)
		}
	}
}

func handleBroadcastMessages(msgChan <-chan body_types.Message) {
	for msg := range msgChan {
		switch msg.ID {
		case demmonAPI.DiscoverMessageID:
			var tableMsg tableMessageWithHost

			err := mapstructure.Decode(msg.Content, &tableMsg)
			if err != nil {
				log.Panic(err)
			}

			remoteTable.updateFromDiscoverMsg(tableMsg.Host, tableMsg.DiscoverMsg)
		}
	}
}

func registerDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling request in registerDeployment handler")

	deploymentID := internalUtils.ExtractPathVar(r, deploymentIDPathVar)
	reqBody := api.RegisterDeploymentRequestBody{}

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	deploymentDTO := reqBody.Deployment

	deployment := &api.Deployment{
		ID:    deploymentID,
		Ports: deploymentDTO.Ports,
	}

	_, ok := sTable.getDeployment(deploymentID)
	if ok {
		w.WriteHeader(http.StatusConflict)

		return
	}

	newTableEntry := &api.DeploymentsTableEntryDTO{
		Deployment: deployment,
		Instances:  map[string]*api.Instance{},
		MaxHops:    maxHops,
	}

	sTable.addDeployment(deploymentID, newTableEntry)

	log.Debugf("added deployment %s", deploymentID)

	clientsManager.AddDeployment(deploymentID)
}

func deleteDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling request in deleteDeployment handler")

	deploymentID := internalUtils.ExtractPathVar(r, deploymentIDPathVar)

	_, ok := sTable.getDeployment(deploymentID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	sTable.deleteDeployment(deploymentID)
	redirectServicesMap.Delete(deploymentID)

	log.Debugf("deleted deployment %s", deploymentID)
}

func registerDeploymentInstanceHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling request in registerDeploymentInstance handler")

	deploymentID := internalUtils.ExtractPathVar(r, deploymentIDPathVar)

	_, ok := sTable.getDeployment(deploymentID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	instanceID := internalUtils.ExtractPathVar(r, instanceIDPathVar)
	req := api.RegisterDeploymentInstanceRequestBody{}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	instanceDTO := req

	ok = sTable.deploymentHasInstance(deploymentID, instanceID)
	if ok {
		w.WriteHeader(http.StatusConflict)

		return
	}

	instance := &api.Instance{
		ID:              instanceID,
		DeploymentID:    deploymentID,
		PortTranslation: instanceDTO.PortTranslation,
		Initialized:     instanceDTO.Static,
		Static:          instanceDTO.Static,
		Local:           instanceDTO.Local,
		Hops:            0,
		Host:            myself,
	}

	sTable.addInstance(deploymentID, instanceID, instance)

	log.Debugf("added instance %s to deployment %s", instanceID, deploymentID)
}

func deleteDeploymentInstanceHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling request in deleteDeploymentInstance handler")

	deploymentID := internalUtils.ExtractPathVar(r, deploymentIDPathVar)

	_, ok := sTable.getDeployment(deploymentID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	instanceID := internalUtils.ExtractPathVar(r, instanceIDPathVar)

	_, ok = sTable.getDeploymentInstance(deploymentID, instanceID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	sTable.deleteInstance(instanceID)

	log.Debugf("deleted instance %s from deployment %s", instanceID, deploymentID)
}

func getAllDeploymentsHandler(w http.ResponseWriter, _ *http.Request) {
	log.Debug("handling request in getAllDeployments handler")

	resp := sTable.getAllDeployments()
	internalUtils.SendJSONReplyOK(w, resp)
}

func getAllDeploymentInstancesHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling request in getAllDeploymentInstances handler")

	deploymentID := internalUtils.ExtractPathVar(r, deploymentIDPathVar)

	_, ok := sTable.getDeployment(deploymentID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	resp := sTable.getAllDeploymentInstances(deploymentID)
	internalUtils.SendJSONReplyOK(w, resp)
}

func getDeploymentInstanceHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling request in getDeploymentInstance handler")

	deploymentID := internalUtils.ExtractPathVar(r, deploymentIDPathVar)
	instanceID := internalUtils.ExtractPathVar(r, instanceIDPathVar)

	instance, ok := sTable.getDeploymentInstance(deploymentID, instanceID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	resp := *instance

	internalUtils.SendJSONReplyOK(w, resp)
}

func getInstanceHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling get instance")

	instanceID := internalUtils.ExtractPathVar(r, instanceIDPathVar)

	log.Debugf("attempting to get instance %s", instanceID)

	instance, ok := sTable.getInstance(instanceID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		internalUtils.SendJSONReplyStatus(w, http.StatusNotFound, nil)

		return
	}

	resp := *instance
	internalUtils.SendJSONReplyOK(w, resp)
}

func whoAreYouHandler(w http.ResponseWriter, _ *http.Request) {
	log.Debug("handling whoAreYou request")

	resp := archimedesID

	internalUtils.SendJSONReplyOK(w, resp)
}

func getDeploymentsTableHandler(w http.ResponseWriter, _ *http.Request) {
	var resp api.GetDeploymentsTableResponseBody

	discoverMsg := sTable.toDiscoverMsg()
	if discoverMsg != nil {
		resp = *discoverMsg
	}

	internalUtils.SendJSONReplyOK(w, resp)
}

func resolveHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var reqBody api.ResolveRequestBody

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		log.Errorf("(%s) bad request", reqBody.ID)
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	reqLogger := log.WithField(internalUtils.ReqIDHeaderField, reqBody.ID)
	reqLogger.Level = log.DebugLevel

	defer reqLogger.Debugf("took %f to answer", time.Since(start).Seconds())

	reqLogger.Debugf("got request from %s", reqBody.Location.LatLng().String())

	redirect, targetURL := checkForLoadBalanceRedirections(reqBody.ToResolve.Host)
	if redirect {
		reqLogger.Debugf(
			"redirecting %s to %s to achieve load balancing", reqBody.ToResolve.Host,
			targetURL.Host,
		)
		clientsManager.RemoveFromExploring(reqBody.DeploymentID)
		http.Redirect(w, r, targetURL.String(), http.StatusPermanentRedirect)

		return
	}

	reqLogger.Debugf("redirections %+v", reqBody.Redirects)

	canRedirect := true

	if len(reqBody.Redirects) > 0 {
		lastRedirect := reqBody.Redirects[len(reqBody.Redirects)-1]

		value, ok := redirectingToMe.Load(reqBody.DeploymentID)
		if ok {
			_, ok = value.(redirectingToMeMapValue).Load(lastRedirect)
			canRedirect = !ok

			if ok {
				reqLogger.Debugf("%s is redirecting to me", lastRedirect)
			}
		}
	}

	if canRedirect {
		redirectTo := checkForClosestNodeRedirection(reqBody.DeploymentID, reqBody.Location)

		switch redirectTo.ID {
		case myself.ID:
			reqLogger.Debugf("im the node to redirect to")
		default:
			reqLogger.Debugf("redirecting to %s from %s", redirectTo, reqBody.Location)

			targetURL = url.URL{
				Scheme:      "http",
				Opaque:      "",
				User:        nil,
				Host:        redirectTo.Addr + ":" + strconv.Itoa(archimedes.Port),
				Path:        api.GetResolvePath(),
				RawPath:     "",
				ForceQuery:  false,
				RawQuery:    "",
				Fragment:    "",
				RawFragment: "",
			}

			clientsManager.RemoveFromExploring(reqBody.DeploymentID)
			http.Redirect(w, r, targetURL.String(), http.StatusPermanentRedirect)

			return
		}
	}

	resolved, found := resolveLocally(reqBody.ToResolve, reqLogger)
	if !found {
		if fallback.ID == myself.ID {
			internalUtils.SendJSONReplyStatus(w, http.StatusNotFound, nil)

			return
		}

		reqLogger.Debugf("redirecting to fallback %s", fallback)

		fallbackURL := url.URL{
			Scheme:      "http",
			Opaque:      "",
			User:        nil,
			Host:        fallback.Addr + ":" + strconv.Itoa(archimedes.Port),
			Path:        api.GetResolvePath(),
			RawPath:     "",
			ForceQuery:  false,
			RawQuery:    "",
			Fragment:    "",
			RawFragment: "",
		}

		clientsManager.RemoveFromExploring(reqBody.DeploymentID)
		http.Redirect(w, r, fallbackURL.String(), http.StatusPermanentRedirect)

		return
	}

	reqLogger.Debug("updating num reqs")
	clientsManager.UpdateNumRequests(reqBody.DeploymentID, reqBody.Location)
	reqLogger.Debug("updated num reqs")

	resp := *resolved

	reqLogger.Debug("will remove from exploring")
	clientsManager.RemoveFromExploring(reqBody.DeploymentID)
	reqLogger.Debug("removed from exploring")
	internalUtils.SendJSONReplyOK(w, resp)
}

func checkForLoadBalanceRedirections(hostToResolve string) (redirect bool, targetURL url.URL) {
	redirect = false
	targetURL = url.URL{
		Scheme:      "",
		Opaque:      "",
		User:        nil,
		Host:        "",
		Path:        "",
		RawPath:     "",
		ForceQuery:  false,
		RawQuery:    "",
		Fragment:    "",
		RawFragment: "",
	}

	value, ok := redirectServicesMap.Load(hostToResolve)
	if ok {
		redirectConfig := value.(redirectionsMapValue)
		if !redirectConfig.Done {
			handleRedirection(hostToResolve, redirectConfig)
		}
	}

	return redirect, targetURL
}

func handleRedirection(hostToResolve string, redirectConfig redirectionsMapValue) (redirect bool, targetURL url.URL) {
	current := atomic.AddInt32(&redirectConfig.Current, 1)
	if current <= redirectConfig.Goal {
		redirect, targetURL = true, url.URL{
			Scheme:      "http",
			Opaque:      "",
			User:        nil,
			Host:        redirectConfig.Target + ":" + strconv.Itoa(archimedes.Port),
			Path:        api.GetResolvePath(),
			RawPath:     "",
			ForceQuery:  false,
			RawQuery:    "",
			Fragment:    "",
			RawFragment: "",
		}
	}

	if current == redirectConfig.Goal {
		log.Debugf("completed goal of redirecting %+v clients to %d for deployment %s", redirectConfig.Target,
			redirectConfig.Goal, hostToResolve)

		redirectConfig.Done = true
	}

	return
}

func resolveLocally(toResolve *api.ToResolveDTO, reqLogger *log.Entry) (resolved *api.ResolvedDTO, found bool) {
	found = false

	var (
		deployment *api.Deployment
		instance   *api.Instance
		ok         bool
		remote     = false
	)

	hostUnresolved := toResolve.Host

	deployment, ok = sTable.getDeployment(hostUnresolved)
	if !ok {
		// In case it doesn't have a deployment with this name LOCALLY search LOCALLY for an instance with the given
		// name

		instance, ok = sTable.getInstance(hostUnresolved)
		if ok {
			resolved, found = resolveInstance(toResolve.Port, instance)
			return
		}

		reqLogger.Debugf("no LOCAL deployment or instance for: %s", hostUnresolved)

		// In case it doesn't have neither deployment or an instance with this name LOCALLY search REMOTELY
		deployment, ok = remoteTable.getDeployment(hostUnresolved)
		if !ok {
			// In case it doesn't have a local deployment or instance nor a remote deployment with this name
			// search for a remote instance

			instance, ok = remoteTable.getInstance(hostUnresolved)
			if !ok {
				// This name is nowhere to be found return empty results

				reqLogger.Debugf("no REMOTE deployment or instance for: %s", hostUnresolved)
				return
			}

			resolved, found = resolveInstance(toResolve.Port, instance)
			return
		}

		remote = true
	}

	var randInstance *api.Instance

	if !remote {
		randInstance = resolveDeploymentToInstance(deployment, reqLogger, sTable)
		if randInstance == nil {
			return
		}
	} else {
		randInstance = resolveDeploymentToInstance(deployment, reqLogger, remoteTable)
		if randInstance == nil {
			return
		}
	}

	resolved, found = resolveInstance(toResolve.Port, randInstance)
	if found {
		reqLogger.Debugf("resolved %s:%s to %s:%s", toResolve.Host, toResolve.Port.Port(), resolved.Host,
			resolved.Port)
	}

	return resolved, found
}

func resolveDeploymentToInstance(deployment *api.Deployment, reqLogger *log.Entry,
	resolver deploymentToInstanceResolver) *api.Instance {
	instances := resolver.getAllDeploymentInstances(deployment.ID)
	if len(instances) == 0 {
		reqLogger.Debugf("no instances for deployment %s", deployment.ID)
		return nil
	}

	var randInstance *api.Instance

	randNum := internalUtils.GetRandInt(len(instances))

	for _, instance := range instances {
		if randNum == 0 {
			randInstance = instance
		} else {
			randNum--
		}
	}

	return randInstance
}

func redirectServiceHandler(w http.ResponseWriter, r *http.Request) {
	deploymentID := internalUtils.ExtractPathVar(r, deploymentIDPathVar)

	var req api.RedirectRequestBody

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	_, ok := sTable.deploymentsMap.Load(deploymentID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	_, ok = redirectingToMe.Load(deploymentID)
	if ok {
		w.WriteHeader(http.StatusConflict)

		return
	}

	log.Debugf("redirecting %d clients to %s", req.Amount, req.Target)

	redirectConfig := &redirectedConfig{
		Target:  req.Target,
		Goal:    req.Amount,
		Current: 0,
		Done:    false,
	}

	redirectServicesMap.Store(deploymentID, redirectConfig)
}

func canRedirectToYouHandler(w http.ResponseWriter, r *http.Request) {
	deploymentID := internalUtils.ExtractPathVar(r, deploymentIDPathVar)

	if _, ok := sTable.deploymentsMap.Load(deploymentID); !ok {
		w.WriteHeader(http.StatusConflict)

		return
	}

	if _, ok := redirectServicesMap.Load(deploymentID); ok {
		w.WriteHeader(http.StatusConflict)

		return
	}
}

func willRedirectToYouHandler(w http.ResponseWriter, r *http.Request) {
	deploymentID := internalUtils.ExtractPathVar(r, deploymentIDPathVar)
	nodeID := internalUtils.ExtractPathVar(r, nodeIDPathVar)

	if _, ok := sTable.deploymentsMap.Load(deploymentID); !ok {
		w.WriteHeader(http.StatusConflict)

		return
	}

	if _, ok := redirectServicesMap.Load(deploymentID); ok {
		w.WriteHeader(http.StatusConflict)

		return
	}

	nodesMap := &sync.Map{}
	value, _ := redirectingToMe.LoadOrStore(deploymentID, nodesMap)
	nodesMap = value.(redirectingToMeMapValue)
	nodesMap.Store(nodeID, nil)

	log.Debugf("%s redirecting %s to me", nodeID, deploymentID)
}

func stoppedRedirectingToYouHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentID := internalUtils.ExtractPathVar(r, deploymentIDPathVar)
	nodeID := internalUtils.ExtractPathVar(r, nodeIDPathVar)

	value, ok := redirectingToMe.Load(deploymentID)
	if ok {
		nodesMap := value.(redirectingToMeMapValue)
		nodesMap.Delete(nodeID)
		log.Debugf("%s stopped redirecting %s to me", nodeID, deploymentID)
	}
}

func removeRedirectionHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentID := internalUtils.ExtractPathVar(r, deploymentIDPathVar)
	redirectServicesMap.Delete(deploymentID)
}

func getRedirectedHandler(w http.ResponseWriter, r *http.Request) {
	deploymentID := internalUtils.ExtractPathVar(r, deploymentIDPathVar)

	value, ok := redirectServicesMap.Load(deploymentID)
	if !ok {
		log.Debugf("deployment %s is not being redirected", deploymentID)
		internalUtils.SendJSONReplyStatus(w, http.StatusNotFound, 0)

		return
	}

	redirected := value.(redirectionsMapValue)
	internalUtils.SendJSONReplyOK(w, redirected.Current)
}

func setExploringClientLocationHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentID := internalUtils.ExtractPathVar(r, deploymentIDPathVar)

	var reqBody api.SetExploringClientLocationRequestBody

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		log.Panic(err)
	}

	log.Debugf("set exploring location %v for deployment %s", reqBody, deploymentID)

	clientsManager.SetToExploring(deploymentID, reqBody)
}

func addDeploymentNodeHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentID := internalUtils.ExtractPathVar(r, deploymentIDPathVar)

	var reqBody api.AddDeploymentNodeRequestBody

	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		log.Panic(err)
	}

	log.Debugf("will add node %s for deployment %s", reqBody.Node.ID, deploymentID)

	redirectTargets.add(deploymentID, reqBody.Node, reqBody.Location)

	if reqBody.Exploring {
		exploringNodes.add(deploymentID, reqBody.Node.ID)
	}

	log.Debugf("added node %s for deployment %s (exploring: %t)", reqBody.Node, deploymentID, reqBody.Exploring)
}

func removeDeploymentNodeHandler(_ http.ResponseWriter, r *http.Request) {
	deploymentID := internalUtils.ExtractPathVar(r, deploymentIDPathVar)
	nodeID := internalUtils.ExtractPathVar(r, nodeIDPathVar)

	redirectTargets.delete(deploymentID, nodeID)
	exploringNodes.checkAndDelete(deploymentID, nodeID)

	log.Debugf("deleted node %s for deployment %s", nodeID, deploymentID)
}

func getClientCentroidsHandler(w http.ResponseWriter, r *http.Request) {
	deploymentID := internalUtils.ExtractPathVar(r, deploymentIDPathVar)

	centroids, ok := clientsManager.GetDeploymentClientsCentroids(deploymentID)
	if !ok {
		internalUtils.SendJSONReplyStatus(w, http.StatusNotFound, nil)
	} else {
		centroidTokens := make([]string, len(centroids))
		for i, centroid := range centroids {
			centroidTokens[i] = centroid.ToToken()
		}

		log.Debugf("%s centroids: %+v", deploymentID, centroidTokens)
		internalUtils.SendJSONReplyOK(w, centroids)
	}
}

func resolveInstance(originalPort nat.Port, instance *api.Instance) (*api.ResolvedDTO, bool) {
	portNatResolved, ok := instance.PortTranslation[originalPort]
	if !ok {
		return nil, false
	}

	return &api.ResolvedDTO{
		Host: instance.Host.Addr,
		Port: portNatResolved[0].HostPort,
	}, true
}

func checkForClosestNodeRedirection(deploymentID string, clientLocation s2.CellID) (redirectTo *utils.Node) {
	redirectTo = myself

	var (
		bestDiff = servers.ChordAngleToKM(myLocation.DistanceToCell(s2.CellFromCellID(clientLocation)))
		status   int
	)

	redirectTargets.rangeOver(
		deploymentID, func(node *utils.Node, nodeLocId s2.CellID) bool {
			auxLocation := s2.CellFromCellID(nodeLocId)
			currDiff := servers.ChordAngleToKM(auxLocation.DistanceToCell(s2.CellFromCellID(clientLocation)))
			if currDiff < bestDiff {
				bestDiff = currDiff
				redirectTo = node
			}

			return true
		},
	)

	log.Debugf("best node in vicinity to redirect client from %+v to is %s", clientLocation, redirectTo)

	if redirectTo.ID != myself.ID {
		log.Debugf("will redirect client at %s to %s", clientLocation.ToToken(), redirectTo)

		// TODO this can be change by a load and delete probably
		has := exploringNodes.checkAndDelete(deploymentID, redirectTo.ID)
		if has {
			addr := myself.Addr + ":" + strconv.Itoa(archimedes.Port)
			autoClient := autoFactory.New()

			status = autoClient.SetExploredSuccessfully(addr, deploymentID, redirectTo.ID)
			if status != http.StatusOK {
				log.Errorf("got status %d when setting %s exploration as success", status, redirectTo)
			}
		}
	} else {
		log.Debugf("client is already connected to closest node")
	}

	return redirectTo
}
