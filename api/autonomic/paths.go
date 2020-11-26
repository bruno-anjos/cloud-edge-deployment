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
	DeploymentChildPath  = "/deployments/%s/child"
	DeploymentParentPath = "/deployments/%s/parent"
	IsNodeInVicinityPath = "/vicinity/%s"
	ClosestNodePath      = "/closest"
	VicinityPath         = "/vicinity"
	MyLocationPath       = "/location"
	LoadPath             = "/load/%s"
	ExplorePath          = "/explored/%s/%s"
	BlacklistPath        = "/blacklist/%s"
)

func GetDeploymentsPath() string {
	return PrefixPath + DeploymentsPath
}

func GetDeploymentPath(deploymentId string) string {
	return PrefixPath + fmt.Sprintf(DeploymentPath, deploymentId)
}

func GetDeploymentChildPath(deploymentId string) string {
	return PrefixPath + fmt.Sprintf(DeploymentChildPath, deploymentId)
}

func GetDeploymentParentPath(deploymentId string) string {
	return PrefixPath + fmt.Sprintf(DeploymentParentPath, deploymentId)
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

func GetBlacklistPath(deploymentId string) string {
	return PrefixPath + fmt.Sprintf(BlacklistPath, deploymentId)
}
