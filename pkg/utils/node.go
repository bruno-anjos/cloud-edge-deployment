package utils

import (
	"os"
)

type Node struct {
	ID   string
	Addr string
}

func NodeFromEnv() *Node {
	nodeID, exists := os.LookupEnv(NodeIDEnvVarName)
	if !exists {
		panic("no NODE_ID set in environment")
	}

	nodeIP, exists := os.LookupEnv(NodeIPEnvVarName)
	if !exists {
		panic("no NODE_IP set in environment")
	}

	return NewNode(nodeID, nodeIP)
}

func NewNode(id, addr string) *Node {
	return &Node{
		ID:   id,
		Addr: addr,
	}
}
