package scheduler

import (
	"strconv"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

const (
	Port = 50001
)

var (
	DefaultHostPort = utils.SchedulerServiceName + ":" + strconv.Itoa(Port)
)
