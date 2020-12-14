package utils

import (
	"os"
	"strings"

	"github.com/docker/go-connections/nat"
	log "github.com/sirupsen/logrus"
)

const (
	DeploymentEnvVarName = "DEPLOYMENT_ID"
	InstanceEnvVarName   = "INSTANCE_ID"
	LocationEnvVarName   = "LOCATION"
	ReplicaNumEnvVarName = "REPLICA_NUM"

	NodeIDEnvVarName = "NODE_ID"
	NodeIPEnvVarName = "NODE_IP"
	PortsEnvVarName  = "PORTS"
)

func GetPortMapFromEnvVar() (portMap nat.PortMap) {
	hostIP, ok := os.LookupEnv(NodeIPEnvVarName)
	if !ok {
		log.Panic("No node ip env var.")
	}

	portsString, ok := os.LookupEnv(PortsEnvVarName)
	if !ok {
		log.Panic("No ports map in env.")
	}

	portsSplitted := strings.Split(portsString, ";")
	portMap = nat.PortMap{}

	for _, portSplitted := range portsSplitted {
		mapping := strings.Split(portSplitted, "->")
		containerPort, hostPort := mapping[0], mapping[1]

		hostBinding := nat.PortBinding{
			HostIP:   hostIP,
			HostPort: hostPort,
		}

		portMap[nat.Port(containerPort)] = []nat.PortBinding{hostBinding}
	}

	return
}
