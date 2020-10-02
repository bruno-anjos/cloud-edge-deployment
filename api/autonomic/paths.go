package autonomic

import (
	"fmt"
)

// Paths
const (
	// TODO make this unexported probably

	PrefixPath = "/autonomic"

	ServicesPath         = "/services"
	ServicePath          = "/services/%s"
	ServiceChildPath     = "/services/%s/child/%s"
	ServiceParentPath    = "/services/%s/parent/%s"
	IsNodeInVicinityPath = "/vicinity/%s"
	ClosestNodePath      = "/closest"
	VicinityPath         = "/vicinity"
	MyLocationPath       = "/location"
	LoadPath             = "/load/%s"
)

func GetServicesPath() string {
	return PrefixPath + ServicesPath
}

func GetServicePath(serviceId string) string {
	return PrefixPath + fmt.Sprintf(ServicePath, serviceId)
}

func GetServiceChildPath(serviceId, childId string) string {
	return PrefixPath + fmt.Sprintf(ServiceChildPath, serviceId, childId)
}

func GetServiceParentPath(serviceId, parentId string) string {
	return PrefixPath + fmt.Sprintf(ServiceParentPath, serviceId, parentId)
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

func GetGetLoadForServicePath(serviceId string) string {
	return PrefixPath + fmt.Sprintf(LoadPath, serviceId)
}
