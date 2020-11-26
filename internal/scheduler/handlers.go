package scheduler

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	archimedesHTTPClient "github.com/bruno-anjos/archimedesHTTPClient"
	api "github.com/bruno-anjos/cloud-edge-deployment/api/scheduler"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type (
	typeInstanceToContainerMapKey   = string
	typeInstanceToContainerMapValue = string
)

const (
	stopContainerTimeout = 10

	networkName = "services-network"
)

var (
	deplClient          = deployer.NewDeployerClient(deployer.LocalHostPort)
	dockerClient        *client.Client
	networkId           string
	instanceToContainer sync.Map
	fallback            string
	myself *utils.Node

	stopContainerTimeoutVar = stopContainerTimeout * time.Second
)

func InitHandlers() {
	for {
		var status int
		fallback, status = deplClient.GetFallback()
		if status != http.StatusOK {
			break
		}
	}

	log.SetLevel(log.DebugLevel)

	myself = utils.NodeFromEnv()

	var err error
	dockerClient, err = client.NewEnvClient()
	if err != nil {
		log.Error("unable to create docker client")
		panic(err)
	}

	instanceToContainer = sync.Map{}

	networkConfig := types.NetworkCreate{
		CheckDuplicate: false,
		Attachable:     false,
	}

	networks, err := dockerClient.NetworkList(context.Background(), types.NetworkListOptions{})

	exists := false
	for _, netVal := range networks {
		if netVal.Name == networkName {
			networkId = netVal.ID
			exists = true
			break
		}
	}

	if !exists {
		var resp types.NetworkCreateResponse
		resp, err = dockerClient.NetworkCreate(context.Background(), networkName, networkConfig)
		if err != nil {
			panic(err)
		}

		networkId = resp.ID
		log.Debug("created network with id ", networkId)
	} else {
		log.Debug("network ", networkName, " already exists")
	}

	log.SetLevel(log.InfoLevel)
}

func startInstanceHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling start instance")

	var containerInstance api.ContainerInstanceDTO
	err := json.NewDecoder(r.Body).Decode(&containerInstance)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if containerInstance.DeploymentName == "" || containerInstance.ImageName == "" {
		log.Errorf("invalid container instance: %v", containerInstance)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	go startContainerAsync(&containerInstance)
}

func stopInstanceHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling delete instance")

	instanceId := utils.ExtractPathVar(r, instanceIdPathVar)

	if instanceId == "" {
		log.Errorf("no instance provided", instanceId)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	value, ok := instanceToContainer.Load(instanceId)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	contId := value.(typeInstanceToContainerMapValue)
	go stopContainerAsync(instanceId, contId)
}

func stopAllInstancesHandler(_ http.ResponseWriter, _ *http.Request) {
	deleteAllInstances()
}

func startContainerAsync(containerInstance *api.ContainerInstanceDTO) {
	portBindings := generatePortBindings(containerInstance.Ports)

	// Create container and get containers id in response
	instanceId := containerInstance.DeploymentName + "-" + utils.RandomString(10) + "-" + hostname

	log.Debugf("instance %s has following portBindings: %+v", instanceId, portBindings)

	deploymentIdEnvVar := utils.DeploymentEnvVarName + "=" + containerInstance.DeploymentName
	instanceIdEnvVar := utils.InstanceEnvVarName + "=" + instanceId
	fallbackEnvVar := archimedesHTTPClient.FallbackEnvVar + "=" + fallback
	nodeEnvVar := "NODE" + "=" + hostname

	// TODO CHANGE THIS TO USE THE ACTUAL LOCATION TOKEN
	locationEnvVar := utils.LocationEnvVarName + "=" + "0c"

	envVars := []string{deploymentIdEnvVar, instanceIdEnvVar, locationEnvVar, fallbackEnvVar, nodeEnvVar}
	envVars = append(envVars, containerInstance.EnvVars...)

	reader, err := dockerClient.ImagePull(context.Background(), containerInstance.ImageName, types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}

	_, err = ioutil.ReadAll(reader)
	if err != nil {
		panic(err)
	}

	err = reader.Close()
	if err != nil {
		panic(err)
	}

	containerConfig := container.Config{
		Cmd:   containerInstance.Command,
		Env:   envVars,
		Image: containerInstance.ImageName,
	}

	hostConfig := container.HostConfig{
		NetworkMode:  "bridge",
		PortBindings: portBindings,
	}

	networkConfig := network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"bridge": {
				Aliases:   []string{instanceId},
				NetworkID: networkId,
			},
		},
	}

	cont, err := dockerClient.ContainerCreate(context.Background(), &containerConfig, &hostConfig, &networkConfig,
		instanceId)
	if err != nil {
		panic(err)
	}

	log.Debugf("connecting %s to network %s", cont.ID, networkId)

	// Add container instance to deployer
	status := deplClient.RegisterDeploymentInstance(containerInstance.DeploymentName, instanceId,
		containerInstance.Static, portBindings, true)
	if status != http.StatusOK {
		err = dockerClient.ContainerStop(context.Background(), cont.ID, &stopContainerTimeoutVar)
		if err != nil {
			log.Error(err)
		}
		log.Fatalf("got status code %d while adding instances to deployer", status)
		return
	}

	// Spin container up
	err = dockerClient.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	if err != nil {
		panic(err)
	}

	instanceToContainer.Store(instanceId, cont.ID)

	log.Debugf("container %s started for instance %s", cont.ID, instanceId)
}

func stopContainerAsync(instanceId, contId string) {
	err := dockerClient.ContainerStop(context.Background(), contId, &stopContainerTimeoutVar)
	if err != nil {
		panic(err)
	}

	log.Debugf("deleted instance %s corresponding to container %s", instanceId, contId)
}

func deleteAllInstances() {
	log.Debugf("stopping all containers")

	instanceToContainer.Range(func(key, value interface{}) bool {
		instanceId := value.(typeInstanceToContainerMapKey)
		contId := value.(typeInstanceToContainerMapValue)

		log.Debugf("stopping instance %s (container %s)", instanceId, contId)

		err := dockerClient.ContainerStop(context.Background(), contId, &stopContainerTimeoutVar)
		if err != nil {
			log.Warnf("error while stopping instance %s (container %s): %s", instanceId, contId, err)
			return true
		}

		return true
	})

	containers, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	for _, containerListed := range containers {
		for _, name := range containerListed.Names {
			if strings.Contains(name, hostname) {
				log.Warnf("deleting orphan container %s", containerListed.ID)
				err = dockerClient.ContainerStop(context.Background(), containerListed.ID, &stopContainerTimeoutVar)
				if err != nil {
					log.Errorf("error stopping orphan container %s: %s", containerListed.ID, err)
				}
				break
			}
		}
	}
}

func getFreePort(protocol string) string {
	switch protocol {
	case utils.TCP:
		addr, err := net.ResolveTCPAddr(utils.TCP, "0.0.0.0:0")
		if err != nil {
			panic(err)
		}

		l, err := net.ListenTCP(utils.TCP, addr)
		if err != nil {
			panic(err)
		}

		defer func() {
			err = l.Close()
			if err != nil {
				panic(err)
			}
		}()

		natPort, err := nat.NewPort(utils.TCP, strconv.Itoa(l.Addr().(*net.TCPAddr).Port))
		if err != nil {
			panic(err)
		}

		return natPort.Port()
	case utils.UDP:
		addr, err := net.ResolveUDPAddr(utils.UDP, "0.0.0.0:0")
		if err != nil {
			panic(err)
		}

		l, err := net.ListenUDP(utils.UDP, addr)
		if err != nil {
			panic(err)
		}

		defer func() {
			err = l.Close()
			if err != nil {
				panic(err)
			}
		}()

		natPort, err := nat.NewPort(utils.UDP, strconv.Itoa(l.LocalAddr().(*net.UDPAddr).Port))
		if err != nil {
			panic(err)
		}

		return natPort.Port()
	default:
		panic(errors.Errorf("invalid port protocol: %s", protocol))
	}
}

func generatePortBindings(containerPorts nat.PortSet) (portMap nat.PortMap) {
	portMap = nat.PortMap{}

	for containerPort := range containerPorts {

		hostBinding := nat.PortBinding{
			HostIP:   myself.Addr,
			HostPort: getFreePort(containerPort.Proto()),
		}
		portMap[containerPort] = []nat.PortBinding{hostBinding}
	}

	return
}
