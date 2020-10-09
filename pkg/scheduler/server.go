package scheduler

import (
	"strconv"

	publicUtils "github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
)

const (
	Port = 50001
)

var (
	DefaultHostPort = publicUtils.SchedulerServiceName + ":" + strconv.Itoa(Port)
)
