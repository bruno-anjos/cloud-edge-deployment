package deployer

import (
	"strconv"
)

const (
	Port                = 50002
	DeployerServiceName = "deployer"
)

var (
	DefaultHostPort = DeployerServiceName + ":" + strconv.Itoa(Port)
)
