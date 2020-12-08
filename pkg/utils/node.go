package utils

import (
	"os"
)

type Node struct {
	Id   string
	Addr string
}

func NodeFromEnv() *Node {
	nodeId, exists := os.LookupEnv(nodeIdEnvVarName)
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
