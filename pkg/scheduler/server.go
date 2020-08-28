package scheduler

import (
	"strconv"
)

const (
	Port                 = 50001
	SchedulerServiceName = "scheduler"
)

var (
	DefaultHostPort = SchedulerServiceName + ":" + strconv.Itoa(Port)
)
