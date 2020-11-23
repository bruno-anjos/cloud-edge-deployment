package utils

const (
	DeploymentEnvVarName = "DEPLOYMENT_ID"
	InstanceEnvVarName   = "INSTANCE_ID"
	LocationEnvVarName   = "LOCATION"
)

const (
	TCP string = "tcp"
	UDP string = "udp"
)

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
