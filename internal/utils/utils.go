package utils

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/golang/geo/s1"
)

const (
	DeploymentEnvVarName = "DEPLOYMENT_ID"
	InstanceEnvVarName   = "INSTANCE_ID"
	LocationEnvVarName   = "LOCATION"
	NodeIdEnvVarName     = "NODE_ID"
	NodeIPEnvVarName     = "NODE_IP"
)

const (
	TCP string = "tcp"
	UDP string = "udp"

	LocalHost = ""
)

type Node struct {
	Id   string
	Addr string
}

func NodeFromEnv() *Node {
	nodeId, exists := os.LookupEnv(NodeIdEnvVarName)
	if !exists {
		panic("no NODE_ID set in environment")
	}

	nodeIP, exists := os.LookupEnv(NodeIPEnvVarName)
	if !exists {
		panic("no NODE_IP set in environment")
	}

	return NewNode(nodeId, nodeIP)
}

func NewNode(id, addr string) *Node {
	return &Node{
		Id:   id,
		Addr: addr,
	}
}

func GetLocalHostPort(port int) string{
	nodeIP, exists := os.LookupEnv(NodeIPEnvVarName)
	if !exists {
		panic("no NODE_IP set in environment")
	}

	return fmt.Sprintf("%s:%d", nodeIP, port)
}

func RandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

const (
	EarthRadius = 6_378
)

func ChordAngleToKM(angle s1.ChordAngle) float64 {
	return angle.Angle().Radians() * EarthRadius
}
