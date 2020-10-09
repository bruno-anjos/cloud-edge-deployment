package autonomic

import (
	"strconv"

	publicUtils "github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
)

const (
	Port = 50003
)

var (
	DefaultHostPort = publicUtils.AutonomicServiceName + ":" + strconv.Itoa(Port)
)
