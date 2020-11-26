package autonomic

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

const (
	Port = 50003
)

var (
	LocalHostPort = utils.GetLocalHostPort(Port)
)