package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	archimedesHTTPClient "github.com/bruno-anjos/archimedesHTTPClient"
	api "github.com/bruno-anjos/cloud-edge-deployment/api/scheduler"
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
	stopContainerTimeout = 10

	envVarFormatString = "%s=%s"

	instanceIdEnvVarReplace = "$instance"
)

var (
	deplClient deployer.Client

	dockerClient        *client.Client
	instanceToContainer sync.Map
	fallback            *internalUtils.Node
	myself              *internalUtils.Node

	stopContainerTimeoutVar = stopContainerTimeout * time.Second
)

func InitServer(deplFactory deployer.ClientFactory) {
	deplClient = deplFactory.New(internalUtils.DeployerLocalHostPort)

	for {
		var status int
		fallback, status = deplClient.GetFallback()
		if status != http.StatusOK {
			break
		}
	}

	log.SetLevel(log.DebugLevel)

	myself = internalUtils.NodeFromEnv()

	var err error
	dockerClient, err = client.NewEnvClient()
	if err != nil {
		log.Error("unable to create docker client")
		panic(err)
	}

	instanceToContainer = sync.Map{}

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

	instanceId := internalUtils.ExtractPathVar(r, instanceIdPathVar)

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
	instanceId := containerInstance.DeploymentName + "-" + internalUtils.RandomString(10) + "-" + myself.Id

	log.Debugf("instance %s has following portBindings: %+v", instanceId, portBindings)

	deploymentIdEnvVar := fmt.Sprintf(envVarFormatString, utils.DeploymentEnvVarName, containerInstance.DeploymentName)
	instanceIdEnvVar := fmt.Sprintf(envVarFormatString, utils.InstanceEnvVarName, instanceId)
	fallbackEnvVar := fmt.Sprintf(envVarFormatString, archimedesHTTPClient.FallbackEnvVar, fallback.Addr)
	nodeIpEnvVar := fmt.Sprintf(envVarFormatString, utils.NodeIPEnvVarName, myself.Addr)
	replicaNumEnvVar := fmt.Sprintf(envVarFormatString, utils.ReplicaNumEnvVarName,
		strconv.Itoa(containerInstance.ReplicaNumber))

	// TODO CHANGE THIS TO USE THE ACTUAL LOCATION TOKEN
	locationEnvVar := fmt.Sprintf(envVarFormatString, utils.LocationEnvVarName, "0c")

	for idx, envVar := range containerInstance.EnvVars {
		if envVar == instanceIdEnvVarReplace {
			containerInstance.EnvVars[idx] = instanceId
		}
	}

	envVars := []string{deploymentIdEnvVar, instanceIdEnvVar, locationEnvVar, fallbackEnvVar, nodeIpEnvVar,
		replicaNumEnvVar}
	envVars = append(envVars, containerInstance.EnvVars...)

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

	var (
		containerConfig container.Config
	)
	if !hasImage {
		log.Debugf("pulling image %s", imageName)

		var reader io.ReadCloser
		reader, err = dockerClient.ImagePull(context.Background(), containerInstance.ImageName, types.ImagePullOptions{})
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

		imageName = containerInstance.ImageName
	} else {
		log.Debugf("image %s already exists locally", imageName)
	}

	containerConfig = container.Config{
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

	cont, err := dockerClient.ContainerCreate(context.Background(), &containerConfig, &hostConfig, nil,
		instanceId)
	if err != nil {
		panic(err)
	}

	if len(cont.Warnings) > 0 {
		for _, warn := range cont.Warnings {
			log.Warn(warn)
		}
	}

	// Add container instance to deployer
	status := deplClient.RegisterDeploymentInstance(containerInstance.DeploymentName, instanceId,
		containerInstance.Static, portBindings, true)
	if status != http.StatusOK {
		err = dockerClient.ContainerStop(context.Background(), cont.ID, &stopContainerTimeoutVar)
		if err != nil {
			log.Error(err)
		}
		log.Panicf("got status code %d while adding instances to deployer", status)
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
			if strings.Contains(name, myself.Id) {
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
	case internalUtils.TCP:
		addr, err := net.ResolveTCPAddr(internalUtils.TCP, "0.0.0.0:0")
		if err != nil {
			panic(err)
		}

		l, err := net.ListenTCP(internalUtils.TCP, addr)
		if err != nil {
			panic(err)
		}

		defer func() {
			err = l.Close()
			if err != nil {
				panic(err)
			}
		}()

		natPort, err := nat.NewPort(internalUtils.TCP, strconv.Itoa(l.Addr().(*net.TCPAddr).Port))
		if err != nil {
			panic(err)
		}

		return natPort.Port()
	case internalUtils.UDP:
		addr, err := net.ResolveUDPAddr(internalUtils.UDP, "0.0.0.0:0")
		if err != nil {
			panic(err)
		}

		l, err := net.ListenUDP(internalUtils.UDP, addr)
		if err != nil {
			panic(err)
		}

		defer func() {
			err = l.Close()
			if err != nil {
				panic(err)
			}
		}()

		natPort, err := nat.NewPort(internalUtils.UDP, strconv.Itoa(l.LocalAddr().(*net.UDPAddr).Port))
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
