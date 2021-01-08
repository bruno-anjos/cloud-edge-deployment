package autonomic

import (
	"fmt"
)

// Paths.
const (
	PrefixPath = "/autonomic"

	DeploymentsPath              = "/deployments"
	DeploymentPath               = "/deployments/%s"
	DeploymentChildPath          = "/deployments/%s/child"
	DeploymentChildWithChildPath = "/deployments/%s/child/%s"
	DeploymentParentPath         = "/deployments/%s/parent"
	IsNodeInVicinityPath         = "/vicinity/%s"
	ClosestNodePath              = "/closest"
	MyLocationPath               = "/location"
	LoadPath                     = "/load/%s"
	ExplorePath                  = "/explored/%s/%s"
	BlacklistPath                = "/blacklist/%s"
	GetIDPath                    = "/getid"
)

func GetDeploymentsPath() string {
	return PrefixPath + DeploymentsPath
}

func GetDeploymentPath(deploymentID string) string {
	return PrefixPath + fmt.Sprintf(DeploymentPath, deploymentID)
}

func GetDeploymentChildPath(deploymentID string) string {
	return PrefixPath + fmt.Sprintf(DeploymentChildPath, deploymentID)
}

func GetDeploymentChildWithChildPath(deploymentID, childID string) string {
	return PrefixPath + fmt.Sprintf(DeploymentChildWithChildPath, deploymentID, childID)
}

func GetDeploymentParentPath(deploymentID string) string {
	return PrefixPath + fmt.Sprintf(DeploymentParentPath, deploymentID)
}

func GetIsNodeInVicinityPath(nodeID string) string {
	return PrefixPath + fmt.Sprintf(IsNodeInVicinityPath, nodeID)
}

func GetClosestNodePath() string {
	return PrefixPath + ClosestNodePath
}

func GetMyLocationPath() string {
	return PrefixPath + MyLocationPath
}

func GetExploredPath(deploymentID, childID string) string {
	return PrefixPath + fmt.Sprintf(ExplorePath, deploymentID, childID)
}

func GetBlacklistPath(deploymentID string) string {
	return PrefixPath + fmt.Sprintf(BlacklistPath, deploymentID)
}
