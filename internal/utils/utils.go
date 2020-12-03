package utils

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/golang/geo/s1"
)

const (
	TCP string = "tcp"
	UDP string = "udp"

	ArchimedesPort = 5000
	AutonomicPort  = 5000
	DeployerPort   = 5000
	SchedulerPort  = 5000
)

type Node struct {
	Id   string
	Addr string
}

var (
	ArchimedesLocalHostPort = GetLocalHostPort(ArchimedesPort)
	AutonomicLocalHostPort  = GetLocalHostPort(AutonomicPort)
	DeployerLocalHostPort   = GetLocalHostPort(DeployerPort)
	SchedulerLocalHostPort  = GetLocalHostPort(SchedulerPort)
)

func NodeFromEnv() *Node {
	nodeId, exists := os.LookupEnv(utils.NodeIdEnvVarName)
	if !exists {
		panic("no NODE_ID set in environment")
	}

	nodeIP, exists := os.LookupEnv(utils.NodeIPEnvVarName)
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

func GetLocalHostPort(port int) string {
	nodeIP, exists := os.LookupEnv(utils.NodeIPEnvVarName)
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
