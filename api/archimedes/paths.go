package archimedes

import (
	"fmt"
)

// Paths
const (
	// TODO make this unexported probably

	PrefixPath = "/archimedes"

	DeploymentsPath             = "/deployments"
	DeploymentPath              = "/deployments/%s"
	DeploymentInstancePath      = "/deployments/%s/%s"
	InstancePath                = "/instances/%s"
	DiscoverPath                = "/discover"
	WhoAreYouPath               = "/who"
	TablePath                   = "/table"
	ResolvePath                 = "/resolve"
	ResolveLocallyPath          = "/resolve/local"
	RedirectPath                = "/deployments/%s/redirect"
	RedirectedPath              = "/deployments/%s/redirected"
	SetResolvingAnswerPath      = "/deployments/asnwer"
	LoadPath                    = "/deployments/%s/load"
	AvgClientLocationPath       = "/deployments/%s/avg_cli_loc"
	ExploringClientLocationPath = "/deployments/%s/exploring_loc"
	AddDeploymentNodePath       = "/deployments/%s/node"
	RemoveDeploymentNodePath    = "/deployments/%s/node/%s"
	RedirectingToYouPath        = "/deployments/%s/redirecting/%s"
)

func GetDeploymentsPath() string {
	return PrefixPath + DeploymentsPath
}

func GetDeploymentPath(deploymentId string) string {
	return PrefixPath + fmt.Sprintf(DeploymentPath, deploymentId)
}

func GetInstancePath(instanceId string) string {
	return PrefixPath + fmt.Sprintf(InstancePath, instanceId)
}

func GetDeploymentInstancePath(deploymentId, instanceId string) string {
	return PrefixPath + fmt.Sprintf(DeploymentInstancePath, deploymentId, instanceId)
}

func GetResolvePath() string {
	return PrefixPath + ResolvePath
}

func GetResolveLocallyPath() string {
	return PrefixPath + ResolveLocallyPath
}

func GetRedirectPath(deploymentId string) string {
	return PrefixPath + fmt.Sprintf(RedirectPath, deploymentId)
}

func GetRedirectedPath(deploymentId string) string {
	return PrefixPath + fmt.Sprintf(RedirectedPath, deploymentId)
}

func GetLoadPath(deploymentId string) string {
	return PrefixPath + fmt.Sprintf(LoadPath, deploymentId)
}

func GetAvgClientLocationPath(deploymentId string) string {
	return PrefixPath + fmt.Sprintf(AvgClientLocationPath, deploymentId)
}

func GetSetExploringClientLocationPath(deploymentId string) string {
	return PrefixPath + fmt.Sprintf(ExploringClientLocationPath, deploymentId)
}

func GetAddDeploymentNodePath(deploymentId string) string {
	return PrefixPath + fmt.Sprintf(AddDeploymentNodePath, deploymentId)
}

func GetRemoveDeploymentNodePath(deploymentId, nodeId string) string {
	return PrefixPath + fmt.Sprintf(RemoveDeploymentNodePath, deploymentId, nodeId)
}

func GetRedirectingToYouPath(deploymentId, nodeId string) string {
	return PrefixPath + fmt.Sprintf(RedirectingToYouPath, deploymentId, nodeId)
}