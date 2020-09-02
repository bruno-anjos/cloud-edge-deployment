package autonomic

import (
	"strconv"
)

const (
	AutonomicServiceName = "autonomic"
	Port                 = 50003
)

var (
	DefaultHostPort = AutonomicServiceName + ":" + strconv.Itoa(Port)
)
