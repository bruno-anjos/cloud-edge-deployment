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

	envVarFormatString = "%s=%s"

	instanceIDEnvVarReplace = "$instance"
)

var (
	deplClient deployer.Client

	dockerClient        *client.Client
	instanceToContainer sync.Map
	fallback            *utils.Node
	myself              *utils.Node

	stopContainerTimeoutVar = stopContainerTimeout
)

func InitServer(deplFactory deployer.ClientFactory) {
	deplClient = deplFactory.New(servers.DeployerLocalHostPort)

	for {
		var status int

		fallback, status = deplClient.GetFallback()
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
	go stopContainerAsync(instanceID, contID)
}

func stopAllInstancesHandler(_ http.ResponseWriter, _ *http.Request) {
	deleteAllInstances()
}

func startContainerAsync(containerInstance *api.ContainerInstanceDTO) {
	portBindings := generatePortBindings(containerInstance.Ports)

	// Create container and get containers id in response
	var instanceID string
	if containerInstance.InstanceName == "" {
		instanceID = containerInstance.DeploymentName + "-" + servers.RandomString(randomChars) + "-" + myself.ID
	} else {
		instanceID = containerInstance.InstanceName
	}

	log.Debugf("instance %s has following portBindings: %+v", instanceID, portBindings)

	deploymentIDEnvVar := fmt.Sprintf(envVarFormatString, utils.DeploymentEnvVarName, containerInstance.DeploymentName)
	instanceIDEnvVar := fmt.Sprintf(envVarFormatString, utils.InstanceEnvVarName, instanceID)
	fallbackEnvVar := fmt.Sprintf(envVarFormatString, archimedesHTTPClient.FallbackEnvVar, fallback.Addr)
	nodeIPEnvVar := fmt.Sprintf(envVarFormatString, utils.NodeIPEnvVarName, myself.Addr)
	replicaNumEnvVar := fmt.Sprintf(envVarFormatString, utils.ReplicaNumEnvVarName,
		strconv.Itoa(containerInstance.ReplicaNumber))

	// TODO CHANGE THIS TO USE THE ACTUAL LOCATION TOKEN
	locationEnvVar := fmt.Sprintf(envVarFormatString, utils.LocationEnvVarName, "0c")

	for idx, envVar := range containerInstance.EnvVars {
		if envVar == instanceIDEnvVarReplace {
			containerInstance.EnvVars[idx] = instanceID
		}
	}

	envVars := []string{
		deploymentIDEnvVar, instanceIDEnvVar, locationEnvVar, fallbackEnvVar, nodeIPEnvVar,
		replicaNumEnvVar,
	}
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

	var containerConfig container.Config

	if !hasImage {
		log.Debugf("pulling image %s", imageName)

		var reader io.ReadCloser

		reader, err = dockerClient.ImagePull(context.Background(), containerInstance.ImageName,
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

		imageName = containerInstance.ImageName
	} else {
		log.Debugf("image %s already exists locally", imageName)
	}

	//nolint:exhaustivestruct
	containerConfig = container.Config{
		Cmd:          containerInstance.Command,
		Env:          envVars,
		Image:        imageName,
		Hostname:     fmt.Sprintf("%s-%d", containerInstance.DeploymentName, containerInstance.ReplicaNumber),
		ExposedPorts: containerInstance.Ports,
	}

	//nolint:exhaustivestruct
	hostConfig := container.HostConfig{
		NetworkMode:  "bridge",
		PortBindings: portBindings,
	}

	cont, err := dockerClient.ContainerCreate(context.Background(), &containerConfig, &hostConfig, nil,
		instanceID)
	if err != nil {
		panic(err)
	}

	if len(cont.Warnings) > 0 {
		for _, warn := range cont.Warnings {
			log.Warn(warn)
		}
	}

	// Add container instance to deployer
	status := deplClient.RegisterDeploymentInstance(containerInstance.DeploymentName, instanceID,
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

	instanceToContainer.Store(instanceID, cont.ID)

	log.Debugf("container %s started for instance %s", cont.ID, instanceID)
}

func stopContainerAsync(instanceID, contID string) {
	err := dockerClient.ContainerStop(context.Background(), contID, &stopContainerTimeoutVar)
	if err != nil {
		log.Panic(err)
	}

	//nolint:exhaustivestruct
	err = dockerClient.ContainerRemove(context.Background(), contID, types.ContainerRemoveOptions{
		//nolint:exhaustivestruct
		Force: true,
	})
	if err != nil {
		log.Panic(err)
	}

	log.Debugf("deleted instance %s corresponding to container %s", instanceID, contID)
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

		defer func() {
			err = l.Close()
			if err != nil {
				panic(err)
			}
		}()

		natPort, err := nat.NewPort(servers.TCP, strconv.Itoa(l.Addr().(*net.TCPAddr).Port))
		if err != nil {
			panic(err)
		}

		return natPort.Port()
	case servers.UDP:
		addr, err := net.ResolveUDPAddr(servers.UDP, "0.0.0.0:0")
		if err != nil {
			panic(err)
		}

		l, err := net.ListenUDP(servers.UDP, addr)
		if err != nil {
			panic(err)
		}

		defer func() {
			err = l.Close()
			if err != nil {
				panic(err)
			}
		}()

		natPort, err := nat.NewPort(servers.UDP, strconv.Itoa(l.LocalAddr().(*net.UDPAddr).Port))
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
