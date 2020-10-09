package deployer

import (
	"strconv"

	publicUtils "github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
)

const (
	Port = 50002
)

var (
	DefaultHostPort = publicUtils.DeployerServiceName + ":" + strconv.Itoa(Port)
)
