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
	GetFallbackIDPath          = "/deployments/fallback"
	HasDeploymentPath          = "/deployments/%s/has"
	PropagateLocationToHorizon = "/deployments/%s/propagate_location"

	DeploymentInstanceAlivePath = "/deployments/%s/%s/alive"
	DeploymentInstancePath      = "/deployments/%s/%s"
	SetReadyPath                = "/set_ready"
)

func GetDeploymentsPath() string {
	return PrefixPath + DeploymentsPath
}

func GetDeploymentPath(deploymentID string) string {
	return PrefixPath + fmt.Sprintf(DeploymentPath, deploymentID)
}

func GetImYourParentPath(deploymentID string) string {
	return PrefixPath + fmt.Sprintf(IAmYourParentPath, deploymentID)
}

func GetParentAlivePath(parentID string) string {
	return PrefixPath + fmt.Sprintf(ParentAlivePath, parentID)
}

func GetDeadChildPath(deploymentID, deadChildID string) string {
	return PrefixPath + fmt.Sprintf(DeadChildPath, deploymentID, deadChildID)
}

func GetDeploymentInstancePath(deploymentID, instanceID string) string {
	return PrefixPath + fmt.Sprintf(DeploymentInstancePath, deploymentID, instanceID)
}

func GetHierarchyTablePath() string {
	return PrefixPath + HierarchyTablePath
}

func GetSetAlternativesPath(nodeID string) string {
	return PrefixPath + fmt.Sprintf(SetAlternativesPath, nodeID)
}

func GetDeploymentInstanceAlivePath(deploymentID, instanceID string) string {
	return PrefixPath + fmt.Sprintf(DeploymentInstanceAlivePath, deploymentID, instanceID)
}

func GetDeploymentChildPath(deploymentID, childID string) string {
	return PrefixPath + fmt.Sprintf(DeploymentChildPath, deploymentID, childID)
}

func GetExtendDeploymentPath(deploymentID string) string {
	return PrefixPath + fmt.Sprintf(ExtendDeploymentToPath, deploymentID)
}

func GetSetGrandparentPath(deploymentID string) string {
	return PrefixPath + fmt.Sprintf(SetGrandparentPath, deploymentID)
}

func GetFallbackPath(deploymentID string) string {
	return PrefixPath + fmt.Sprintf(FallbackPath, deploymentID)
}

func GetGetFallbackIDPath() string {
	return PrefixPath + GetFallbackIDPath
}

func GetHasDeploymentPath(deploymentID string) string {
	return PrefixPath + fmt.Sprintf(HasDeploymentPath, deploymentID)
}

func GetPropagateLocationToHorizonPath(deploymentID string) string {
	return PrefixPath + fmt.Sprintf(PropagateLocationToHorizon, deploymentID)
}

func GetSetReadyPath() string {
	return PrefixPath + SetReadyPath
}
