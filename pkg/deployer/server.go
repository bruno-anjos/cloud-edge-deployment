package deployer

import (
	"strconv"
)

const (
	Port                = 50002
	deployerServiceName = "deployer"
)

var (
	DefaultHostPort = deployerServiceName + ":" + strconv.Itoa(Port)
)
