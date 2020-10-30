package autonomic

import (
	"fmt"
)

// Paths
const (
	// TODO make this unexported probably

	PrefixPath = "/autonomic"

	DeploymentsPath      = "/deployments"
	DeploymentPath       = "/deployments/%s"
	DeploymentChildPath  = "/deployments/%s/child/%s"
	DeploymentParentPath = "/deployments/%s/parent/%s"
	IsNodeInVicinityPath = "/vicinity/%s"
	ClosestNodePath      = "/closest"
	VicinityPath         = "/vicinity"
	MyLocationPath       = "/location"
	LoadPath             = "/load/%s"
	ExplorePath          = "/explored/%s/%s"
	BlacklistPath        = "/blacklist/%s/%s"
)

func GetDeploymentsPath() string {
	return PrefixPath + DeploymentsPath
}

func GetDeploymentPath(deploymentId string) string {
	return PrefixPath + fmt.Sprintf(DeploymentPath, deploymentId)
}

func GetDeploymentChildPath(deploymentId, childId string) string {
	return PrefixPath + fmt.Sprintf(DeploymentChildPath, deploymentId, childId)
}

func GetDeploymentParentPath(deploymentId, parentId string) string {
	return PrefixPath + fmt.Sprintf(DeploymentParentPath, deploymentId, parentId)
}

func GetIsNodeInVicinityPath(nodeId string) string {
	return PrefixPath + fmt.Sprintf(IsNodeInVicinityPath, nodeId)
}

func GetClosestNodePath() string {
	return PrefixPath + ClosestNodePath
}

func GetVicinityPath() string {
	return PrefixPath + VicinityPath
}

func GetMyLocationPath() string {
	return PrefixPath + MyLocationPath
}

func GetExploredPath(deploymentId, childId string) string {
	return PrefixPath + fmt.Sprintf(ExplorePath, deploymentId, childId)
}

func GetBlacklistPath(deploymentId, nodeId string) string {
	return PrefixPath + fmt.Sprintf(BlacklistPath, deploymentId, nodeId)
}
