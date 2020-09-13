package autonomic

import (
	"fmt"
)

// Paths
const (
	// TODO make this unexported probably

	PrefixPath = "/autonomic"

	ServicesPath      = "/services"
	ServicePath       = "/services/%s"
	ServiceChildPath  = "/services/%s/child/%s"
	ServiceParentPath = "/services/%s/parent/%s"
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
