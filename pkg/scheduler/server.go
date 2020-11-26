package scheduler

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

const (
	Port = 50001
)

var (
	LocalHostPort = utils.GetLocalHostPort(Port)
)
