package scheduler

import (
	"strconv"
)

const (
	Port                 = 50001
	schedulerServiceName = "scheduler"
)

var (
	DefaultHostPort = schedulerServiceName + ":" + strconv.Itoa(Port)
)
