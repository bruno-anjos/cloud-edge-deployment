package deployer

import (
	"fmt"
	"strconv"
)

// Paths
const (
	PrefixPath = "/deployer"

	DeploymentsPath = "/deployments"
	DeploymentPath  = "/deployments/%s"

	AddNodePath = "/node"

	WhoAreYouPath = "/who"

	SetAlternativesPath = "/alternatives/%s"

	DeploymentQualityPath = "/deployments/%s/quality"
	DeadChildPath         = "/deployments/%s/deadchild/%s"
	TakeChildPath         = "/deployments/%s/child"
	IAmYourParentPath     = "/deployments/%s/parent"
	HierarchyTablePath    = "/table"
	ParentAlivePath       = "/parent/%s/up"

	// scheduler
	DeploymentInstanceAlivePath = "/deployments/%s/%s/alive"
	DeploymentInstancePath      = "/deployments/%s/%s"
)

const (
	Port = 50002
)

var (
	DeployerServiceName = "deployer"
	DefaultHostPort     = DeployerServiceName + ":" + strconv.Itoa(Port)
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

func GetTakeChildPath(serviceId string) string {
	return PrefixPath + fmt.Sprintf(TakeChildPath, serviceId)
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
