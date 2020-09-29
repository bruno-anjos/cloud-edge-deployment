package deployer

import (
	"strconv"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

const (
	Port = 50002
)

var (
	DefaultHostPort = utils.DeployerServiceName + ":" + strconv.Itoa(Port)
)
