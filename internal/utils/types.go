package utils

import (
	"math"
)

const (
	ServiceEnvVarName  = "SERVICE_ID"
	InstanceEnvVarName = "INSTANCE_ID"
)

const (
	TCP string = "tcp"
	UDP string = "udp"
)

type Location struct {
	X float64
	Y float64
}

func (l *Location) CalcDist(l2 *Location) float64 {
	dX := l2.X - l.X
	dY := l2.Y - l.Y
	return math.Sqrt(math.Pow(dX, 2) + math.Pow(dY, 2))
}

type Node struct {
	Id   string
	Addr string
}

func NewNode(id, addr string) *Node {
	return &Node{
		Id:   id,
		Addr: addr,
	}
}
