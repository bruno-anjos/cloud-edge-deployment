package deployer

import (
	"fmt"
)

// Paths
const (
	PrefixPath = "/deployer"

	DeploymentsPath = "/deployments"
	DeploymentPath  = "/deployments/%s"

	WhoAreYouPath = "/who"

	SetAlternativesPath = "/alternatives/%s"

	DeadChildPath              = "/deployments/%s/deadchild/%s"
	IAmYourParentPath          = "/deployments/%s/parent"
	HierarchyTablePath         = "/table"
	ParentAlivePath            = "/parent/%s/up"
	DeploymentChildPath        = "/deployments/%s/child/%s"
	ExtendDeploymentToPath     = "/deployments/%s/extend"
	SetGrandparentPath         = "/deployments/%s/grandparent"
	FallbackPath               = "/deployments/%s/fallback"
	GetFallbackIdPath          = "/deployments/fallback"
	HasDeploymentPath          = "/deployments/%s/has"
	PropagateLocationToHorizon = "/deployments/%s/propagate_location"

	DeploymentInstanceAlivePath = "/deployments/%s/%s/alive"
	DeploymentInstancePath      = "/deployments/%s/%s"
)

func GetDeploymentsPath() string {
	return PrefixPath + DeploymentsPath
}

func GetDeploymentPath(deploymentId string) string {
	return PrefixPath + fmt.Sprintf(DeploymentPath, deploymentId)
}

func GetImYourParentPath(deploymentId string) string {
	return PrefixPath + fmt.Sprintf(IAmYourParentPath, deploymentId)
}

func GetParentAlivePath(parentId string) string {
	return PrefixPath + fmt.Sprintf(ParentAlivePath, parentId)
}

func GetDeadChildPath(deploymentId, deadChildId string) string {
	return PrefixPath + fmt.Sprintf(DeadChildPath, deploymentId, deadChildId)
}

func GetDeploymentInstancePath(deploymentId, instanceId string) string {
	return PrefixPath + fmt.Sprintf(DeploymentInstancePath, deploymentId, instanceId)
}

func GetHierarchyTablePath() string {
	return PrefixPath + HierarchyTablePath
}

func GetSetAlternativesPath(nodeId string) string {
	return PrefixPath + fmt.Sprintf(SetAlternativesPath, nodeId)
}

func GetDeploymentInstanceAlivePath(deploymentId, instanceId string) string {
	return PrefixPath + fmt.Sprintf(DeploymentInstanceAlivePath, deploymentId, instanceId)
}

func GetDeploymentChildPath(deploymentId, childId string) string {
	return PrefixPath + fmt.Sprintf(DeploymentChildPath, deploymentId, childId)
}

func GetExtendDeploymentPath(deploymentId string) string {
	return PrefixPath + fmt.Sprintf(ExtendDeploymentToPath, deploymentId)
}

func GetSetGrandparentPath(deploymentId string) string {
	return PrefixPath + fmt.Sprintf(SetGrandparentPath, deploymentId)
}

func GetFallbackPath(deploymentId string) string {
	return PrefixPath + fmt.Sprintf(FallbackPath, deploymentId)
}

func GetGetFallbackIdPath() string {
	return PrefixPath + GetFallbackIdPath
}

func GetHasDeploymentPath(deploymentId string) string {
	return PrefixPath + fmt.Sprintf(HasDeploymentPath, deploymentId)
}

func GetPropagateLocationToHorizonPath(deploymentId string) string {
	return PrefixPath + fmt.Sprintf(PropagateLocationToHorizon, deploymentId)
}
