package archimedes

import (
	"strconv"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

const (
	Port = 50000
)

var (
	DefaultHostPort = utils.ArchimedesServiceName + ":" + strconv.Itoa(Port)
)
