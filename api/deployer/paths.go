package deployer

import (
	"fmt"
)

// Paths
const (
	PrefixPath = "/deployer"

	DeploymentsPath = "/deployments"
	DeploymentPath  = "/deployments/%s"

	AddNodePath = "/node"

	WhoAreYouPath = "/who"

	SetAlternativesPath = "/alternatives/%s"

	DeploymentQualityPath     = "/deployments/%s/quality"
	DeadChildPath             = "/deployments/%s/deadchild/%s"
	IAmYourParentPath         = "/deployments/%s/parent"
	HierarchyTablePath        = "/table"
	ParentAlivePath           = "/parent/%s/up"
	DeploymentChildPath       = "/deployments/%s/child/%s"
	MigrateDeploymentPath     = "/deployments/%s/migrate"
	ExtendServiceToPath       = "/deployments/%s/extend/%s"
	ShortenServiceFromPath    = "/deployments/%s/shorten/%s"
	CanTakeChildPath          = "/deployments/%s/can_child/%s"
	SetGrandparentPath        = "/deployments/%s/grandparent"
	CanTakeParentPath         = "/deployments/%s/can_parent/%s"
	FallbackPath              = "/deployments/%s/fallback"
	ResolveUpTheTreePath      = "/deployments/%s/resolve_up"
	StartResolveUpTheTreePath = "/deployments/%s/start_resolve_up"
	RedirectDownTheTreePath   = "/deployments/%s/redirect_down"

	// scheduler
	DeploymentInstanceAlivePath = "/deployments/%s/%s/alive"
	DeploymentInstancePath      = "/deployments/%s/%s"
)

func GetDeploymentsPath() string {
	return PrefixPath + DeploymentsPath
}

func GetServicePath(serviceId string) string {
	return PrefixPath + fmt.Sprintf(DeploymentPath, serviceId)
}

func GetExpandTreePath(serviceId string) string {
	return PrefixPath + fmt.Sprintf(DeploymentQualityPath, serviceId)
}

func GetImYourParentPath(serviceId string) string {
	return PrefixPath + fmt.Sprintf(IAmYourParentPath, serviceId)
}

func GetAddNodePath() string {
	return PrefixPath + AddNodePath
}

func GetParentAlivePath(parentId string) string {
	return PrefixPath + fmt.Sprintf(ParentAlivePath, parentId)
}

func GetDeadChildPath(serviceId, deadChildId string) string {
	return PrefixPath + fmt.Sprintf(DeadChildPath, serviceId, deadChildId)
}

func GetServiceInstancePath(serviceId, instanceId string) string {
	return PrefixPath + fmt.Sprintf(DeploymentInstancePath, serviceId, instanceId)
}

func GetHierarchyTablePath() string {
	return PrefixPath + HierarchyTablePath
}

func GetSetAlternativesPath(nodeId string) string {
	return PrefixPath + fmt.Sprintf(SetAlternativesPath, nodeId)
}

func GetWhoAreYouPath() string {
	return PrefixPath + WhoAreYouPath
}

func GetServiceInstanceAlivePath(serviceId, instanceId string) string {
	return PrefixPath + fmt.Sprintf(DeploymentInstanceAlivePath, serviceId, instanceId)
}

func GetDeploymentChildPath(deploymentId, childId string) string {
	return PrefixPath + fmt.Sprintf(DeploymentChildPath, deploymentId, childId)
}

func GetMigrateDeploymentPath(deploymentId string) string {
	return PrefixPath + fmt.Sprintf(MigrateDeploymentPath, deploymentId)
}

func GetExtendServicePath(serviceId, targetId string) string {
	return PrefixPath + fmt.Sprintf(ExtendServiceToPath, serviceId, targetId)
}

func GetShortenServicePath(serviceId, targetId string) string {
	return PrefixPath + fmt.Sprintf(ShortenServiceFromPath, serviceId, targetId)
}

func GetCanTakeChildPath(deploymentId, childId string) string {
	return PrefixPath + fmt.Sprintf(CanTakeChildPath, deploymentId, childId)
}

func GetCanTakeParentPath(deploymentId, parentId string) string {
	return PrefixPath + fmt.Sprintf(CanTakeParentPath, deploymentId, parentId)
}

func GetSetGrandparentPath(deploymentId string) string {
	return PrefixPath + fmt.Sprintf(SetGrandparentPath, deploymentId)
}

func GetFallbackPath(deploymentId string) string {
	return PrefixPath + fmt.Sprintf(FallbackPath, deploymentId)
}

func GetResolveUpTheTreePath(deploymentId string) string {
	return PrefixPath + fmt.Sprintf(ResolveUpTheTreePath, deploymentId)
}

func GetStartResolveUpTheTreePath(deploymentId string) string {
	return PrefixPath + fmt.Sprintf(StartResolveUpTheTreePath, deploymentId)
}

func GetRedirectDownTheTreePath(deploymentId string) string {
	return PrefixPath + fmt.Sprintf(RedirectDownTheTreePath, deploymentId)
}
