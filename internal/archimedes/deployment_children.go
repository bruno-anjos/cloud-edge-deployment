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

func (nd *nodesPerDeployment) add(deploymentID string, node *utils.Node, location s2.CellID) {
	nodes := &deploymentNodes{}

	value, _ := nd.nodes.LoadOrStore(deploymentID, nodes)
	nodes = value.(nodesPerDeploymentMapValue)
	nodes.Store(node.ID, &deploymentNodesMapValue{Node: node, Location: location})
}

func (nd *nodesPerDeployment) delete(deploymentID, nodeID string) {
	value, ok := nd.nodes.Load(deploymentID)
	if !ok {
		return
	}

	nodes := value.(nodesPerDeploymentMapValue)
	nodes.Delete(nodeID)
}

func (nd *nodesPerDeployment) rangeOver(deploymentID string, f func(node *utils.Node, nodeLoc s2.CellID) bool) {
	value, ok := nd.nodes.Load(deploymentID)
	if !ok {
		return
	}

	nodes := value.(nodesPerDeploymentMapValue)
	nodes.Range(func(key, value interface{}) bool {
		depValue := value.(*deploymentNodesMapValue)

		return f(depValue.Node, depValue.Location)
	})
}
