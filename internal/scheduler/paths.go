package scheduler

import (
	"fmt"
)

// Paths
const (
	PrefixPath = "/scheduler"

	InstancesPath = "/instances"
	InstancePath  = "/instances/%s"
)

func GetInstancesPath() string {
	return PrefixPath + InstancesPath
}

func GetInstancePath(instanceId string) string {
	return PrefixPath + fmt.Sprintf(InstancePath, instanceId)
}
