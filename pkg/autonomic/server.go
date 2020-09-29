package autonomic

import (
	"strconv"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

const (
	Port = 50003
)

var (
	DefaultHostPort = utils.AutonomicServiceName + ":" + strconv.Itoa(Port)
)
