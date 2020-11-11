package archimedes

import (
	"sync"

	"github.com/golang/geo/s2"
)

type (
	deploymentNodesMapKey = string
	deploymentNodesMapValue = s2.CellID

	deploymentNodes struct {
		sync.Map
	}

	nodesPerDeploymentMapValue = *deploymentNodes

	nodesPerDeployment struct {
		nodes sync.Map
	}
)

func (nd *nodesPerDeployment) add(deploymentId, nodeId string, location s2.CellID) {
	nodes := &deploymentNodes{}

	value, _ := nd.nodes.LoadOrStore(deploymentId, nodes)
	nodes = value.(nodesPerDeploymentMapValue)
	nodes.Store(nodeId, location)
}

func (nd *nodesPerDeployment) delete(deploymentId, nodeId string) {
	value, ok := nd.nodes.Load(deploymentId)
	if !ok {
		return
	}

	nodes := value.(nodesPerDeploymentMapValue)
	nodes.Delete(nodeId)
}

func (nd *nodesPerDeployment) rangeOver(deploymentId string, f func(nodeId string, nodeLoc s2.CellID) bool) {
	value, ok := nd.nodes.Load(deploymentId)
	if !ok {
		return
	}

	nodes := value.(nodesPerDeploymentMapValue)
	nodes.Range(func(key, value interface{}) bool {
		return f(key.(deploymentNodesMapKey), value.(deploymentNodesMapValue))
	})
}
