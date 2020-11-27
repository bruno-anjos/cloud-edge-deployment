package archimedes

import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

const (
	Port = 50000
)

var (
	LocalHostPort = utils.GetLocalHostPort(Port)
)
