package autonomic

import (
	"strconv"
)

const (
	autonomicServiceName = "autonomic"
	Port                 = 50003
)

var (
	DefaultHostPort = autonomicServiceName + ":" + strconv.Itoa(Port)
)
