package deployer

import (
	"strconv"
)

const (
	Port        = 50002
	ServiceName = "deployer"
)

var (
	DefaultHostPort = ServiceName + ":" + strconv.Itoa(Port)
)
