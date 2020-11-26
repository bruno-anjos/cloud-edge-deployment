package deployer

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

const (
	Port = 50002
)

var (
	LocalHostPort = utils.GetLocalHostPort(Port)
)
