package archimedes

import (
	"fmt"
)

// Paths
const (
	// TODO make this unexported probably

	PrefixPath = "/archimedes"

	ServicesPath        = "/services"
	ServicePath         = "/services/%s"
	ServiceInstancePath = "/services/%s/%s"
	InstancePath        = "/instances/%s"
	DiscoverPath        = "/discover"
	WhoAreYouPath       = "/who"
	TablePath           = "/table"
	ResolvePath         = "/resolve"
	RedirectPath        = "/services/%s/redirect"
	RedirectedPath      = "/services/%s/redirected"
)

func GetServicesPath() string {
	return PrefixPath + ServicesPath
}

func GetServicePath(serviceId string) string {
	return PrefixPath + fmt.Sprintf(ServicePath, serviceId)
}

func GetInstancePath(instanceId string) string {
	return PrefixPath + fmt.Sprintf(InstancePath, instanceId)
}

func GetServiceInstancePath(serviceId, instanceId string) string {
	return PrefixPath + fmt.Sprintf(ServiceInstancePath, serviceId, instanceId)
}

func GetResolvePath() string {
	return PrefixPath + ResolvePath
}

func GetRedirectPath(serviceId string) string {
	return PrefixPath + fmt.Sprintf(RedirectPath, serviceId)
}

func GetRedirectedPath(serviceId string) string {
	return PrefixPath + fmt.Sprintf(RedirectedPath, serviceId)
}
