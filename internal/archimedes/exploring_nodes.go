package archimedes

import (
	"sync"
)

type (
	deploymentExplorers struct {
		sync.Map
	}

	explorersPerDeploymentMapValue = *deploymentExplorers

	explorersPerDeployment struct {
		explorers sync.Map
	}
)

func (ed *explorersPerDeployment) add(deploymentID, explorerID string) {
	explorers := &deploymentExplorers{}

	value, _ := ed.explorers.LoadOrStore(deploymentID, explorers)
	explorers = value.(explorersPerDeploymentMapValue)
	explorers.Store(explorerID, nil)
}

func (ed *explorersPerDeployment) checkAndDelete(deploymentID, explorerID string) (has bool) {
	value, ok := ed.explorers.Load(deploymentID)
	if !ok {
		return
	}

	explorers := value.(explorersPerDeploymentMapValue)
	_, has = explorers.LoadAndDelete(explorerID)

	return
}
