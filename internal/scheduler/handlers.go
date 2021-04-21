package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	archimedesHTTPClient "github.com/bruno-anjos/archimedesHTTPClient"
	api "github.com/bruno-anjos/cloud-edge-deployment/api/scheduler"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/servers"
	internalUtils "github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
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
	retryTimeout         = 2 * time.Second
	stopContainerTimeout = 10 * time.Second
	retryPortBinding     = 2 * time.Second

	envVarFormatString   = "%s=%s"
	portBindingFmtString = "%s->%s"

	instanceIDEnvVarReplace = "$instance"
)

var (
	deplClient deployer.Client

	dockerClient        *client.Client
	instanceToContainer sync.Map
	fallback            *utils.Node
	myself              *utils.Node
	location            string

	stopContainerTimeoutVar = stopContainerTimeout

	getPortLock = sync.Mutex{}
	usedPorts   = map[string]interface{}{}
	nodeNum     string
)

func InitServer(deplFactory deployer.ClientFactory) {
	log.SetLevel(log.DebugLevel)

	deplClient = deplFactory.New()

	for {
		var status int

		fallback, status = deplClient.GetFallback(servers.DeployerLocalHostPort)
		if status == http.StatusOK {
			break
		}

		time.Sleep(retryTimeout)
	}

	log.SetLevel(log.DebugLevel)

	log.Debugf("fallback: %+v", fallback)

	myself = utils.NodeFromEnv()

	var err error

	dockerClient, err = client.NewEnvClient()
	if err != nil {
		log.Error("unable to create docker client")
		panic(err)
	}

	instanceToContainer = sync.Map{}

	var ok bool

	location, ok = os.LookupEnv(utils.LocationEnvVarName)
	if !ok {
		log.Panic("no location env var")
	}

	log.Debugf("Node at location %s", location)

	nodeNum, ok = os.LookupEnv(utils.NodeNumEnvVarName)
	if !ok {
		log.Panic("env var NODE_NUM missing")
	}
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

	requestBody := api.StopInstanceRequestBody{}

	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		log.Panic(err)
	}

	instanceID := internalUtils.ExtractPathVar(r, instanceIDPathVar)

	if instanceID == "" {
		log.Errorf("no instance provided %s", instanceID)
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	value, ok := instanceToContainer.Load(instanceID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)

		return
	}

	contID := value.(typeInstanceToContainerMapValue)
	go stopContainerAsync(instanceID, contID, requestBody.URL, requestBody.RemovePath)
}

func stopAllInstancesHandler(_ http.ResponseWriter, _ *http.Request) {
	deleteAllInstances()
}

func getEnvVars(containerInstance *api.ContainerInstanceDTO, instanceID string,
	portBindings nat.PortMap) (envVars []string) {
	portBindingsEnvVar := make([]string, 0, len(portBindings))

	for containerPort, hostPorts := range portBindings {
		if len(hostPorts) != 1 {
			log.Panicf("host ports (%+v) for container port %s", hostPorts, containerPort)
		}

		portBindingsEnvVar = append(portBindingsEnvVar, fmt.Sprintf(portBindingFmtString, containerPort,
			hostPorts[0].HostPort))
	}

	deploymentIDEnvVar := fmt.Sprintf(envVarFormatString, utils.DeploymentEnvVarName, containerInstance.DeploymentName)
	instanceIDEnvVar := fmt.Sprintf(envVarFormatString, utils.InstanceEnvVarName, instanceID)
	fallbackEnvVar := fmt.Sprintf(envVarFormatString, archimedesHTTPClient.FallbackEnvVar, fallback.Addr)
	nodeIPEnvVar := fmt.Sprintf(envVarFormatString, utils.NodeIPEnvVarName, myself.Addr)
	portsEnvVar := fmt.Sprintf(envVarFormatString, utils.PortsEnvVarName, strings.Join(portBindingsEnvVar, ";"))
	replicaNumEnvVar := fmt.Sprintf(envVarFormatString, utils.ReplicaNumEnvVarName,
		strconv.Itoa(containerInstance.ReplicaNumber))
	locationEnvVar := fmt.Sprintf(envVarFormatString, utils.LocationEnvVarName, location)
	nodeIDEnvVar := fmt.Sprintf(envVarFormatString, utils.NodeIDEnvVarName, myself.ID)
	nodeNumEnvVar := fmt.Sprintf(envVarFormatString, utils.NodeNumEnvVarName, nodeNum)

	for idx, envVar := range containerInstance.EnvVars {
		if envVar == instanceIDEnvVarReplace {
			containerInstance.EnvVars[idx] = instanceID
		}
	}

	envVars = append(envVars, deploymentIDEnvVar, instanceIDEnvVar, locationEnvVar, fallbackEnvVar, nodeIPEnvVar,
		replicaNumEnvVar, portsEnvVar, nodeIDEnvVar, nodeNumEnvVar)
	envVars = append(envVars, containerInstance.EnvVars...)

	return envVars
}

func startContainerAsync(containerInstance *api.ContainerInstanceDTO) {
	success := false

	var (
		portBindings nat.PortMap
		instanceID   string
		contID       string
	)
	for !success {
		// This can fail because the OS might give us a freeport that has not yet been fully released by docker

		portBindings = generatePortBindings(containerInstance.Ports)

		// Create container and get containers id in response
		if containerInstance.InstanceName == "" {
			instanceID = containerInstance.DeploymentName + "-" + servers.RandomString(randomChars) + "-" + myself.ID
		} else {
			instanceID = containerInstance.InstanceName
		}

		log.Debugf("instance %s has following portBindings: %+v", instanceID, portBindings)

		envVars := getEnvVars(containerInstance, instanceID, portBindings)

		images, err := dockerClient.ImageList(context.Background(), types.ImageListOptions{})
		if err != nil {
			panic(err)
		}

		splits := strings.Split(containerInstance.ImageName, "/")[1:]

		var imageName string

		if splits[0] == "library" {
			imageName = splits[1]
		} else {
			imageName = splits[0] + "/" + splits[1]
		}

		log.Debugf("imagename: %s", imageName)

		hasImage := false

		for _, image := range images {
			if image.RepoTags[0] == imageName {
				log.Debugf("%s matches %s", image.RepoTags[0], imageName)

				hasImage = true
			}
		}

		if !hasImage {
			pullImage(containerInstance, imageName)
			imageName = containerInstance.ImageName
		} else {
			log.Debugf("image %s already exists locally", imageName)
		}

		//nolint:exhaustivestruct
		containerConfig := container.Config{
			Cmd:          containerInstance.Command,
			Env:          envVars,
			Image:        imageName,
			Hostname:     fmt.Sprintf("%s-%d", containerInstance.DeploymentName, containerInstance.ReplicaNumber),
			ExposedPorts: containerInstance.Ports,
		}

		hostConfig := container.HostConfig{
			NetworkMode:  "bridge",
			PortBindings: portBindings,
		}

		// nolint:exhaustivestruct
		err = dockerClient.ContainerRemove(context.Background(), instanceID, types.ContainerRemoveOptions{
			Force: true,
		})

		if err != nil {
			log.Infof(err.Error())
		}

		cont, err := dockerClient.ContainerCreate(context.Background(), &containerConfig, &hostConfig, nil,
			instanceID)
		if err != nil {
			panic(err)
		}

		contID = cont.ID

		if len(cont.Warnings) > 0 {
			for _, warn := range cont.Warnings {
				log.Warn(warn)
			}
		}

		// Add container instance to deployer
		status := deplClient.RegisterDeploymentInstance(servers.DeployerLocalHostPort, containerInstance.DeploymentName,
			instanceID, containerInstance.Static, portBindings, true)
		if status != http.StatusOK {
			err = dockerClient.ContainerStop(context.Background(), contID, &stopContainerTimeoutVar)
			if err != nil {
				log.Error(err)
			}

			log.Warn("got status code %d while adding instances for deployment %d to deployer",
				containerInstance.DeploymentName, status)

			return
		}

		// Spin container up
		err = dockerClient.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
		if err != nil {
			log.Warn(err)
			continue
		} else {
			success = true
		}

		instanceToContainer.Store(instanceID, cont.ID)

		log.Debugf("container %s started for instance %s", cont.ID, instanceID)
	}
}

func pullImage(containerInstance *api.ContainerInstanceDTO, imageName string) {
	log.Debugf("pulling image %s", imageName)

	reader, err := dockerClient.ImagePull(context.Background(), containerInstance.ImageName,
		types.ImagePullOptions{})
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
}

func stopContainerAsync(instanceID, contID, url, removePath string) {
	if removePath != "" {
		log.Infof("url: %s, path: %s", url, removePath)

		httpClient := http.Client{}
		req := internalUtils.BuildRequest(http.MethodGet, url, removePath, nil)

		_, err := httpClient.Do(req)
		if err != nil {
			log.Error(err)
		}
	}

	err := dockerClient.ContainerStop(context.Background(), contID, &stopContainerTimeoutVar)
	if err != nil {
		log.Panic(err)
	}

	// log.Debugf("Removing %s", contID)
	//
	// err = dockerClient.ContainerRemove(context.Background(), contID, types.ContainerRemoveOptions{
	// 	RemoveVolumes: true,
	// 	Force:         true,
	// })
	// if err != nil {
	// 	log.Panic(err)
	// }

	log.Debugf("deleted instance %s corresponding to container %s", instanceID, contID)

	// ideally we should remove the used ports in here to free up, but this shouldn't be a problem in the short term
}

func deleteAllInstances() {
	log.Debugf("stopping all containers")

	instanceToContainer.Range(func(key, value interface{}) bool {
		instanceID := value.(typeInstanceToContainerMapKey)
		contID := value.(typeInstanceToContainerMapValue)

		log.Debugf("stopping instance %s (container %s)", instanceID, contID)

		err := dockerClient.ContainerStop(context.Background(), contID, &stopContainerTimeoutVar)
		if err != nil {
			log.Warnf("error while stopping instance %s (container %s): %s", instanceID, contID, err)

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
			if strings.Contains(name, myself.ID) {
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
	getPortLock.Lock()

	var (
		port        string
		releaseFunc func()
	)

	for {
		switch protocol {
		case servers.TCP:
			addr, err := net.ResolveTCPAddr(servers.TCP, "0.0.0.0:0")
			if err != nil {
				panic(err)
			}

			l, err := net.ListenTCP(servers.TCP, addr)
			if err != nil {
				panic(err)
			}

			releaseFunc = func() {
				err = l.Close()
				if err != nil {
					panic(err)
				}
			}

			natPort, err := nat.NewPort(servers.TCP, strconv.Itoa(l.Addr().(*net.TCPAddr).Port))
			if err != nil {
				panic(err)
			}

			port = natPort.Port()
		case servers.UDP:
			addr, err := net.ResolveUDPAddr(servers.UDP, "0.0.0.0:0")
			if err != nil {
				panic(err)
			}

			l, err := net.ListenUDP(servers.UDP, addr)
			if err != nil {
				panic(err)
			}

			releaseFunc = func() {
				err = l.Close()
				if err != nil {
					panic(err)
				}
			}

			natPort, err := nat.NewPort(servers.UDP, strconv.Itoa(l.LocalAddr().(*net.UDPAddr).Port))
			if err != nil {
				panic(err)
			}

			port = natPort.Port()
		default:
			panic(errors.Errorf("invalid port protocol: %s", protocol))
		}

		if _, ok := usedPorts[port]; !ok {
			usedPorts[port] = nil
			break
		}

		<-time.After(retryPortBinding)
	}

	releaseFunc()
	getPortLock.Unlock()

	return port
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
