package archimedes

import (
	"fmt"
)

// Paths.
const (
	PrefixPath = "/archimedes"

	DeploymentsPath             = "/deployments"
	DeploymentPath              = "/deployments/%s"
	DeploymentInstancePath      = "/deployments/%s/%s"
	InstancePath                = "/instances/%s"
	WhoAreYouPath               = "/who"
	TablePath                   = "/table"
	ResolvePath                 = "/resolve"
	RedirectPath                = "/deployments/%s/redirect"
	RedirectedPath              = "/deployments/%s/redirected"
	SetResolvingAnswerPath      = "/deployments/asnwer"
	ExploringClientLocationPath = "/deployments/%s/exploring_loc"
	AddDeploymentNodePath       = "/deployments/%s/node"
	RemoveDeploymentNodePath    = "/deployments/%s/node/%s"
	RedirectingToYouPath        = "/deployments/%s/redirecting/%s"
)

func GetDeploymentsPath() string {
	return PrefixPath + DeploymentsPath
}

func GetDeploymentPath(deploymentID string) string {
	return PrefixPath + fmt.Sprintf(DeploymentPath, deploymentID)
}

func GetInstancePath(instanceID string) string {
	return PrefixPath + fmt.Sprintf(InstancePath, instanceID)
}

func GetDeploymentInstancePath(deploymentID, instanceID string) string {
	return PrefixPath + fmt.Sprintf(DeploymentInstancePath, deploymentID, instanceID)
}

func GetResolvePath() string {
	return PrefixPath + ResolvePath
}

func GetRedirectPath(deploymentID string) string {
	return PrefixPath + fmt.Sprintf(RedirectPath, deploymentID)
}

func GetRedirectedPath(deploymentID string) string {
	return PrefixPath + fmt.Sprintf(RedirectedPath, deploymentID)
}

func GetSetExploringClientLocationPath(deploymentID string) string {
	return PrefixPath + fmt.Sprintf(ExploringClientLocationPath, deploymentID)
}

func GetAddDeploymentNodePath(deploymentID string) string {
	return PrefixPath + fmt.Sprintf(AddDeploymentNodePath, deploymentID)
}

func GetRemoveDeploymentNodePath(deploymentID, nodeID string) string {
	return PrefixPath + fmt.Sprintf(RemoveDeploymentNodePath, deploymentID, nodeID)
}

func GetRedirectingToYouPath(deploymentID, nodeID string) string {
	return PrefixPath + fmt.Sprintf(RedirectingToYouPath, deploymentID, nodeID)
}
