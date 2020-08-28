package scheduler

import (
	"fmt"
	"strconv"
)

// Paths
const (
	PrefixPath = "/scheduler"

	InstancesPath = "/instances"
	InstancePath  = "/instances/%s"
)

const (
	Port = 50001
)

var (
	SchedulerServiceName = "scheduler"
	DefaultHostPort      = SchedulerServiceName + ":" + strconv.Itoa(Port)
)

func GetInstancesPath() string {
	return PrefixPath + InstancesPath
}

func GetInstancePath(instanceId string) string {
	return PrefixPath + fmt.Sprintf(InstancePath, instanceId)
}
