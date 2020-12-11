package archimedes

import (
	"sync"

	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/golang/geo/s2"
)

type (
	deploymentNodesMapValue struct {
		Location s2.CellID
		Node     *utils.Node
	}

	deploymentNodes struct {
		sync.Map
	}

	nodesPerDeploymentMapValue = *deploymentNodes

	nodesPerDeployment struct {
		nodes sync.Map
	}
)

func (nd *nodesPerDeployment) add(deploymentId string, node *utils.Node, location s2.CellID) {
	nodes := &deploymentNodes{}

	value, _ := nd.nodes.LoadOrStore(deploymentId, nodes)
	nodes = value.(nodesPerDeploymentMapValue)
	nodes.Store(node.Id, &deploymentNodesMapValue{Node: node, Location: location})
}

func (nd *nodesPerDeployment) delete(deploymentId, nodeId string) {
	value, ok := nd.nodes.Load(deploymentId)
	if !ok {
		return
	}

	nodes := value.(nodesPerDeploymentMapValue)
	nodes.Delete(nodeId)
}

func (nd *nodesPerDeployment) rangeOver(deploymentId string, f func(node *utils.Node, nodeLoc s2.CellID) bool) {
	value, ok := nd.nodes.Load(deploymentId)
	if !ok {
		return
	}

	nodes := value.(nodesPerDeploymentMapValue)
	nodes.Range(func(key, value interface{}) bool {
		depValue := value.(*deploymentNodesMapValue)
		return f(depValue.Node, depValue.Location)
	})
}
