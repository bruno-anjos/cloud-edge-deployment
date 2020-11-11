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

func (ed *explorersPerDeployment) add(deploymentId, explorerId string) {
	explorers := &deploymentExplorers{}

	value, _ := ed.explorers.LoadOrStore(deploymentId, explorers)
	explorers = value.(explorersPerDeploymentMapValue)
	explorers.Store(explorerId, nil)
}

func (ed *explorersPerDeployment) checkAndDelete(deploymentId, explorerId string) (has bool) {
	value, ok := ed.explorers.Load(deploymentId)
	if !ok {
		return
	}

	explorers := value.(explorersPerDeploymentMapValue)
	_, has = explorers.LoadAndDelete(explorerId)
	return
}